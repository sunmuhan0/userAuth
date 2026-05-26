package client

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	pb "ttuser/auth-client/auth"
	configclient "ttuser/config-client/client"
	"ttuser/pkg/trace"
)

const (
	// 默认gRPC地址，后续从配置中心获取
	defaultAddr = "localhost:9090"
	// CA 证书路径，后续从配置中心获取
	defaultCACert = "../certs/ca.pem"
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
	addr := defaultAddr
	caCert := defaultCACert

	// 尝试从配置中心获取
	cfg := configclient.DefaultConfig()
	cfg.ServiceName = "proc"
	cc := configclient.New(cfg)
	if err := cc.Start(0); err == nil {
		var authConf struct {
			Addr   string `json:"addr"`
			CACert string `json:"ca_cert"`
		}
		if err := cc.Get("auth-client", &authConf); err == nil {
			addr = authConf.Addr
			caCert = authConf.CACert
			fmt.Println("[AuthClient] config loaded from config-center")
		}
	}

	// 加载 CA 证书用于验证服务端
	creds, err := credentials.NewClientTLSFromFile(caCert, "localhost")
	if err != nil {
		return fmt.Errorf("failed to load CA certificate from %s: %w", caCert, err)
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
