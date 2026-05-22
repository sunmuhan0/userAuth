package manager

import (
	"context"

	pb "ttuser/auth-client/auth"
	authclient "ttuser/auth-client/client"
)

// AuthManager 封装对 auth-server 的 gRPC 调用
type AuthManager struct {
	AuthClient *authclient.AuthClient `inject:"authClient"`
}

// Login 调用auth-server登录接口
func (m *AuthManager) Login(ctx context.Context, username, password string) (*pb.LoginResponse, error) {
	return m.AuthClient.Login(ctx, &pb.LoginRequest{
		Username: username,
		Password: password,
	})
}

// Logout 调用auth-server注销接口（同时废弃access和refresh token）
func (m *AuthManager) Logout(ctx context.Context, accessToken, refreshToken string) (*pb.LogoutResponse, error) {
	return m.AuthClient.Logout(ctx, &pb.LogoutRequest{
		Token:        accessToken,
		RefreshToken: refreshToken,
	})
}

// GetUserInfo 调用auth-server获取用户信息
func (m *AuthManager) GetUserInfo(ctx context.Context, userID string) (*pb.GetUserInfoResponse, error) {
	return m.AuthClient.GetUserInfo(ctx, &pb.GetUserInfoRequest{
		UserId: userID,
	})
}

// UpdateUserInfo 调用auth-server更新用户信息
func (m *AuthManager) UpdateUserInfo(ctx context.Context, userID, nickname, email, avatar string) (*pb.UpdateUserInfoResponse, error) {
	return m.AuthClient.UpdateUserInfo(ctx, &pb.UpdateUserInfoRequest{
		UserId:   userID,
		Nickname: nickname,
		Email:    email,
		Avatar:   avatar,
	})
}

// ValidateToken 调用auth-server验证token
func (m *AuthManager) ValidateToken(ctx context.Context, token string) (*pb.ValidateTokenResponse, error) {
	return m.AuthClient.ValidateToken(ctx, &pb.ValidateTokenRequest{
		Token: token,
	})
}

// RefreshToken 调用auth-server续签token
func (m *AuthManager) RefreshToken(ctx context.Context, refreshToken string) (*pb.RefreshTokenResponse, error) {
	return m.AuthClient.RefreshToken(ctx, &pb.RefreshTokenRequest{
		RefreshToken: refreshToken,
	})
}
