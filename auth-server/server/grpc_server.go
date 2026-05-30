package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/teou/inji"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"

	pb "ttuser/auth-client/auth"
	"ttuser/auth-server/internal/model"
	"ttuser/auth-server/internal/service"
	"ttuser/auth-server/pkg/interceptor"
	configclient "ttuser/config-client/client"
	"ttuser/pkg/log"
	pnacos "ttuser/pkg/nacos"
)

const (
	defaultPort = 9090
)

// AuthGRPCServer gRPC服务端实现
type AuthGRPCServer struct {
	pb.UnimplementedAuthServiceServer
	AuthService *service.AuthServiceImpl `inject:"authService"`
	server      *grpc.Server
	httpServer  *http.Server
	Registrar   pnacos.IServiceRegistrar `inject:"serviceRegistrar"`
	ServerName  string                   `inject:"serverName"`
}

// SetRegistrar 设置服务注册器
func (s *AuthGRPCServer) SetRegistrar(r pnacos.IServiceRegistrar) {
	s.Registrar = r
}

// Run 启动gRPC服务（带TLS）
func (s *AuthGRPCServer) Run() error {
	port := defaultPort
	if v, ok := inji.Find("serverPort"); ok {
		vStr, ok := v.(string)
		if ok {
			if p, err := strconv.Atoi(vStr); err == nil {
				port = p
			}
		}
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	// 从配置中心加载TLS证书
	svc := s.ServerName

	var certsConf struct {
		Cert string `json:"cert"`
		Key  string `json:"key"`
	}
	if err := configclient.LoadFile(svc, "certs.json", &certsConf); err != nil {
		return fmt.Errorf("failed to load certs config: %w", err)
	}
	certPair, tlsErr := tls.X509KeyPair([]byte(certsConf.Cert), []byte(certsConf.Key))
	if tlsErr != nil {
		fmt.Printf("[auth-server] TLS cert error: %v, running without TLS\n", tlsErr)
	}

	var grpcOpts []grpc.ServerOption
	if tlsErr == nil {
		creds := credentials.NewServerTLSFromCert(&certPair)
		grpcOpts = append(grpcOpts, grpc.Creds(creds))
	}

	grpc_prometheus.EnableHandlingTimeHistogram()
	grpcOpts = append(grpcOpts,
		grpc.ChainUnaryInterceptor(
			grpc_prometheus.UnaryServerInterceptor,
			interceptor.UnaryServerTraceInterceptor(),
		),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     15 * time.Minute,
			MaxConnectionAge:      30 * time.Minute,
			MaxConnectionAgeGrace: 5 * time.Minute,
			Time:                  10 * time.Second,
			Timeout:               5 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.MaxRecvMsgSize(4*1024*1024),
	)
	s.server = grpc.NewServer(grpcOpts...)
	grpc_prometheus.Register(s.server)
	pb.RegisterAuthServiceServer(s.server, s)

	if s.Registrar != nil {
		if err := s.Registrar.Start(); err != nil {
			fmt.Printf("[auth-server] failed to register service: %v\n", err)
		}
	}

	// 启动HTTP metrics端口（gRPC port + 100）
	httpPort := port + 100
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", httpPort),
		Handler: mux,
	}
	go func() {
		fmt.Printf("[auth-server] HTTP metrics listening on :%d\n", httpPort)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("[auth-server] HTTP metrics server error: %v\n", err)
		}
	}()

	fmt.Printf("[auth-server] gRPC listening on :%d (TLS)\n", port)
	return s.server.Serve(lis)
}

// Stop 优雅停止
func (s *AuthGRPCServer) Stop() {
	if s.server != nil {
		s.server.GracefulStop()
	}
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.httpServer.Shutdown(ctx)
	}
	if s.Registrar != nil {
		s.Registrar.Close()
	}
}

// Register 注册RPC
func (s *AuthGRPCServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	if s.AuthService == nil {
		return nil, status.Error(codes.Internal, "auth service not initialized")
	}
	if req.Username == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "username and password are required")
	}
	user, err := s.AuthService.Register(ctx, req.Username, req.Password, req.Nickname, req.Email)
	if err != nil {
		if err == service.ErrUserAlreadyExists {
			return nil, status.Error(codes.AlreadyExists, "username already exists")
		}
		log.Error(ctx, "register failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &pb.RegisterResponse{User: userToProto(user)}, nil
}

// Login 登录RPC
func (s *AuthGRPCServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if s.AuthService == nil {
		return nil, status.Error(codes.Internal, "auth service not initialized")
	}
	if req.Username == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "username and password are required")
	}
	accessToken, refreshToken, expiresAt, user, err := s.AuthService.Login(ctx, req.Username, req.Password)
	if err != nil {
		log.Warn(ctx, "login failed", "error", err)
		return nil, status.Error(codes.Unauthenticated, "invalid username or password")
	}
	return &pb.LoginResponse{
		Token:        accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		User:         userToProto(user),
	}, nil
}

// Logout 注销RPC
func (s *AuthGRPCServer) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	if s.AuthService == nil {
		return nil, status.Error(codes.Internal, "auth service not initialized")
	}
	err := s.AuthService.Logout(ctx, req.Token, req.RefreshToken)
	if err != nil {
		log.Error(ctx, "logout failed", "error", err)
		return nil, status.Error(codes.Internal, "logout failed")
	}
	return &pb.LogoutResponse{Success: true}, nil
}

// GetUserInfo 获取用户信息RPC
func (s *AuthGRPCServer) GetUserInfo(ctx context.Context, req *pb.GetUserInfoRequest) (*pb.GetUserInfoResponse, error) {
	if s.AuthService == nil {
		return nil, status.Error(codes.Internal, "auth service not initialized")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	user, err := s.AuthService.GetUserInfo(ctx, req.UserId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	return &pb.GetUserInfoResponse{User: userToProto(user)}, nil
}

// UpdateUserInfo 更新用户信息RPC
func (s *AuthGRPCServer) UpdateUserInfo(ctx context.Context, req *pb.UpdateUserInfoRequest) (*pb.UpdateUserInfoResponse, error) {
	if s.AuthService == nil {
		return nil, status.Error(codes.Internal, "auth service not initialized")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	user, err := s.AuthService.UpdateUserInfo(ctx, req.UserId, req.Nickname, req.Email, req.Avatar)
	if err != nil {
		log.Error(ctx, "update user info failed", "error", err)
		return nil, status.Error(codes.Internal, "update failed")
	}
	return &pb.UpdateUserInfoResponse{User: userToProto(user)}, nil
}

// ValidateToken 验证Token RPC
func (s *AuthGRPCServer) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	if req.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}
	claims, err := s.AuthService.ValidateToken(ctx, req.Token)
	if err != nil {
		return &pb.ValidateTokenResponse{
			Valid:   false,
			Message: err.Error(),
		}, nil
	}
	return &pb.ValidateTokenResponse{
		Valid:    true,
		UserId:   claims.UserID,
		Username: claims.Username,
	}, nil
}

// RefreshToken 续签Token RPC
func (s *AuthGRPCServer) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	if req.RefreshToken == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh_token is required")
	}
	accessToken, refreshToken, expiresAt, err := s.AuthService.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	return &pb.RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

func userToProto(u *model.User) *pb.UserInfo {
	if u == nil {
		return nil
	}
	return &pb.UserInfo{
		Id:        u.ID,
		Username:  u.Username,
		Nickname:  u.Nickname,
		Email:     u.Email,
		Avatar:    u.Avatar,
		CreatedAt: u.CreatedAt.Unix(),
		UpdatedAt: u.UpdatedAt.Unix(),
	}
}
