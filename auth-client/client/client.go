package client

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pb "ttuser/auth-client/auth"
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
	addr := defaultAddr     // TODO: 后续从配置中心获取
	caCert := defaultCACert // TODO: 后续从配置中心获取

	// 加载 CA 证书用于验证服务端
	creds, err := credentials.NewClientTLSFromFile(caCert, "localhost")
	if err != nil {
		return fmt.Errorf("failed to load CA certificate from %s: %w", caCert, err)
	}

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(creds))
	if err != nil {
		return fmt.Errorf("failed to connect to auth-server at %s: %w", addr, err)
	}
	c.conn = conn
	c.IAuthServiceClient = IAuthServiceClient{pb.NewAuthServiceClient(conn)}
	fmt.Printf("[AuthClient] connected to auth-server at %s (TLS)\n", addr)
	return nil
}
