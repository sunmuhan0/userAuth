package server

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"ttuser/auth-server/internal/model"
	"ttuser/auth-server/internal/service"
	pb "ttuser/auth-client/auth"
)

// 默认gRPC监听端口
const defaultPort = 9090

// AuthGRPCServer gRPC服务端实现
type AuthGRPCServer struct {
	pb.UnimplementedAuthServiceServer
	AuthService *service.AuthServiceImpl `inject:"authService"`
	server      *grpc.Server
}

// Run 启动gRPC服务（不命名Start，避免inji自动调用）
func (s *AuthGRPCServer) Run() error {
	port := defaultPort // TODO: 后续从配置中心获取
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.server = grpc.NewServer()
	pb.RegisterAuthServiceServer(s.server, s)

	fmt.Printf("[auth-server] gRPC listening on :%d\n", port)
	return s.server.Serve(lis)
}

// Stop 优雅停止
func (s *AuthGRPCServer) Stop() {
	if s.server != nil {
		s.server.GracefulStop()
	}
}

// Login 登录RPC
func (s *AuthGRPCServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req.Username == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "username and password are required")
	}
	accessToken, refreshToken, expiresAt, user, err := s.AuthService.Login(ctx, req.Username, req.Password)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
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
	err := s.AuthService.Logout(ctx, req.Token, req.RefreshToken)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.LogoutResponse{Success: true}, nil
}

// GetUserInfo 获取用户信息RPC
func (s *AuthGRPCServer) GetUserInfo(ctx context.Context, req *pb.GetUserInfoRequest) (*pb.GetUserInfoResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	user, err := s.AuthService.GetUserInfo(ctx, req.UserId)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return &pb.GetUserInfoResponse{User: userToProto(user)}, nil
}

// UpdateUserInfo 更新用户信息RPC
func (s *AuthGRPCServer) UpdateUserInfo(ctx context.Context, req *pb.UpdateUserInfoRequest) (*pb.UpdateUserInfoResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	user, err := s.AuthService.UpdateUserInfo(ctx, req.UserId, req.Nickname, req.Email, req.Avatar)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
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
