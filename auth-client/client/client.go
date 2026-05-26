package client

import (
	"context"
	"crypto/x509"
	"fmt"

	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/teou/inji"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	pb "ttuser/auth-client/auth"
	configclient "ttuser/config-client/client"
	"ttuser/pkg/nacos"
	"ttuser/pkg/trace"
)

const (
	defaultAddr = "localhost:9090"
)

// IAuthServiceClient 内嵌生成的 gRPC client 接口，方便直接调用 RPC 方法
type IAuthServiceClient struct {
	pb.AuthServiceClient
}

// AuthClient 封装gRPC连接（TLS加密）
// 内嵌 IAuthServiceClient，外部可直接调用 Login/Logout/RefreshToken 等方法
// 实现 inji.Startable / inji.Closeable 接口，支持自动注册
type AuthClient struct {
	conn *grpc.ClientConn
	IAuthServiceClient
}

// Start 实现 inji.Startable 接口，inji创建实例后自动调用建连
func (c *AuthClient) Start() error {
	return c.init()
}

// Close 实现 inji.Closeable 接口，inji.Close() 时自动断连
func (c *AuthClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *AuthClient) init() error {
	var authConf struct {
		Addr   string `json:"addr"`
		CACert string `json:"ca_cert"`
	}
	svc := "proc"
	if v, ok := inji.Find("serverName"); ok {
		svc = v.(string)
	}
	if err := configclient.LoadFile(svc, "auth-client.json", &authConf); err != nil {
		return fmt.Errorf("[AuthClient] load auth-client config failed: %w", err)
	}

	cp := x509.NewCertPool()
	cp.AppendCertsFromPEM([]byte(authConf.CACert))
	creds := credentials.NewClientTLSFromCert(cp, "localhost")

	// 优先通过 Nacos 发现 auth-server 地址
	addr := authConf.Addr
	var nacosCfg nacos.Config
	if err := configclient.LoadFile(svc, "nacos.json", &nacosCfg); err == nil {
		namingClient, err := nacos.NewNamingClient(&nacosCfg)
		if err == nil {
			instance, err := namingClient.SelectOneHealthyInstance(vo.SelectOneHealthInstanceParam{
				ServiceName: "auth-server",
				GroupName:   "DEFAULT_GROUP",
				Clusters:    []string{"DEFAULT"},
			})
			if err == nil && instance != nil {
				addr = fmt.Sprintf("%s:%d", instance.Ip, instance.Port)
				fmt.Printf("[AuthClient] discovered auth-server via nacos: %s\n", addr)
			}
			namingClient.CloseClient()
		}
	}
	if addr == "" {
		addr = defaultAddr
	}

	conn, err := grpc.Dial(addr,
		grpc.WithTransportCredentials(creds),
		grpc.WithUnaryInterceptor(unaryClientTraceInterceptor),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to auth-server at %s: %w", addr, err)
	}
	c.conn = conn
	c.IAuthServiceClient = IAuthServiceClient{pb.NewAuthServiceClient(conn)}
	fmt.Printf("[AuthClient] connected to auth-server at %s (TLS)\n", addr)
	return nil
}

// unaryClientTraceInterceptor gRPC客户端拦截器
// 从ctx提取trace_id，写入outgoing metadata传递给服务端
func unaryClientTraceInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	traceID := trace.GetTraceID(ctx)
	if traceID != "" {
		md := metadata.Pairs(trace.MetadataKey, traceID)
		ctx = metadata.NewOutgoingContext(ctx, md)
	}
	return invoker(ctx, method, req, reply, cc, opts...)
}
