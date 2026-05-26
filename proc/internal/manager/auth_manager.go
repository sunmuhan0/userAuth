package manager

import (
	"context"
	"fmt"
	"time"

	pb "ttuser/auth-client/auth"
	authclient "ttuser/auth-client/client"
)

const grpcTimeout = 10 * time.Second

// AuthManager 封装对 auth-server 的 gRPC 调用
type AuthManager struct {
	AuthClient *authclient.AuthClient `inject:"authClient"`
}

func (m *AuthManager) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, grpcTimeout)
}

// Register 调用auth-server注册接口
func (m *AuthManager) Register(ctx context.Context, username, password, nickname, email string) (resp *pb.RegisterResponse, err error) {
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	resp, err = m.AuthClient.Register(ctx, &pb.RegisterRequest{
		Username: username,
		Password: password,
		Nickname: nickname,
		Email:    email,
	})
	if err != nil {
		return nil, fmt.Errorf("auth_manager.Register: %w", err)
	}
	return resp, nil
}

// Login 调用auth-server登录接口
func (m *AuthManager) Login(ctx context.Context, username, password string) (resp *pb.LoginResponse, err error) {
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	resp, err = m.AuthClient.Login(ctx, &pb.LoginRequest{
		Username: username,
		Password: password,
	})
	if err != nil {
		return nil, fmt.Errorf("auth_manager.Login: %w", err)
	}
	return resp, nil
}

// Logout 调用auth-server注销接口（同时废弃access和refresh token）
func (m *AuthManager) Logout(ctx context.Context, accessToken, refreshToken string) (resp *pb.LogoutResponse, err error) {
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	resp, err = m.AuthClient.Logout(ctx, &pb.LogoutRequest{
		Token:        accessToken,
		RefreshToken: refreshToken,
	})
	if err != nil {
		return nil, fmt.Errorf("auth_manager.Logout: %w", err)
	}
	return resp, nil
}

// GetUserInfo 调用auth-server获取用户信息
func (m *AuthManager) GetUserInfo(ctx context.Context, userID string) (resp *pb.GetUserInfoResponse, err error) {
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	resp, err = m.AuthClient.GetUserInfo(ctx, &pb.GetUserInfoRequest{
		UserId: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("auth_manager.GetUserInfo: %w", err)
	}
	return resp, nil
}

// UpdateUserInfo 调用auth-server更新用户信息
func (m *AuthManager) UpdateUserInfo(ctx context.Context, userID, nickname, email, avatar string) (resp *pb.UpdateUserInfoResponse, err error) {
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	resp, err = m.AuthClient.UpdateUserInfo(ctx, &pb.UpdateUserInfoRequest{
		UserId:   userID,
		Nickname: nickname,
		Email:    email,
		Avatar:   avatar,
	})
	if err != nil {
		return nil, fmt.Errorf("auth_manager.UpdateUserInfo: %w", err)
	}
	return resp, nil
}

// ValidateToken 调用auth-server验证token
func (m *AuthManager) ValidateToken(ctx context.Context, token string) (resp *pb.ValidateTokenResponse, err error) {
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	resp, err = m.AuthClient.ValidateToken(ctx, &pb.ValidateTokenRequest{
		Token: token,
	})
	if err != nil {
		return nil, fmt.Errorf("auth_manager.ValidateToken: %w", err)
	}
	return resp, nil
}

// RefreshToken 调用auth-server续签token
func (m *AuthManager) RefreshToken(ctx context.Context, refreshToken string) (resp *pb.RefreshTokenResponse, err error) {
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	resp, err = m.AuthClient.RefreshToken(ctx, &pb.RefreshTokenRequest{
		RefreshToken: refreshToken,
	})
	if err != nil {
		return nil, fmt.Errorf("auth_manager.RefreshToken: %w", err)
	}
	return resp, nil
}
