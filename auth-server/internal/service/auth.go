package service

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"ttuser/auth-server/internal/model"
	"ttuser/auth-server/internal/store"
	"ttuser/auth-server/pkg/token"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrTokenBlacklisted   = errors.New("token has been revoked")
)

// AuthServiceImpl 认证服务实现
type AuthServiceImpl struct {
	UserStore  *store.MemoryUserStore  `inject:"userStore"`
	TokenStore *store.MemoryTokenStore `inject:"tokenStore"`
	TokenMgr   *token.JWTManager      `inject:"tokenManager"`
}

func (s *AuthServiceImpl) Login(ctx context.Context, username, password string) (string, string, int64, *model.User, error) {
	user, err := s.UserStore.GetByUsername(ctx, username)
	if err != nil {
		return "", "", 0, nil, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", "", 0, nil, ErrInvalidCredentials
	}

	// 生成access token（2小时）
	accessToken, expiresAt, err := s.TokenMgr.Generate(user.ID, user.Username)
	if err != nil {
		return "", "", 0, nil, err
	}

	// 生成refresh token（7天）
	refreshToken, _, err := s.TokenMgr.GenerateRefresh(user.ID, user.Username)
	if err != nil {
		return "", "", 0, nil, err
	}

	return accessToken, refreshToken, expiresAt, user, nil
}

func (s *AuthServiceImpl) Logout(ctx context.Context, accessToken, refreshToken string) error {
	// 将access token加入黑名单
	if accessToken != "" {
		claims, err := s.TokenMgr.Parse(accessToken)
		if err == nil {
			_ = s.TokenStore.Add(ctx, accessToken, claims.ExpiresAt.Unix())
		}
	}

	// 将refresh token加入黑名单
	if refreshToken != "" {
		claims, err := s.TokenMgr.ParseRefresh(refreshToken)
		if err == nil {
			_ = s.TokenStore.Add(ctx, refreshToken, claims.ExpiresAt.Unix())
		}
	}

	return nil
}

func (s *AuthServiceImpl) GetUserInfo(ctx context.Context, userID string) (*model.User, error) {
	user, err := s.UserStore.GetByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *AuthServiceImpl) UpdateUserInfo(ctx context.Context, userID, nickname, email, avatar string) (*model.User, error) {
	user, err := s.UserStore.GetByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	if nickname != "" {
		user.Nickname = nickname
	}
	if email != "" {
		user.Email = email
	}
	if avatar != "" {
		user.Avatar = avatar
	}
	if err := s.UserStore.Update(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *AuthServiceImpl) ValidateToken(ctx context.Context, tokenStr string) (*token.Claims, error) {
	blacklisted, err := s.TokenStore.Exists(ctx, tokenStr)
	if err != nil {
		return nil, err
	}
	if blacklisted {
		return nil, ErrTokenBlacklisted
	}
	claims, err := s.TokenMgr.Parse(tokenStr)
	if err != nil {
		return nil, err
	}
	return claims, nil
}

func (s *AuthServiceImpl) RefreshToken(ctx context.Context, refreshTokenStr string) (string, string, int64, error) {
	// 1. 检查refresh token是否在黑名单
	blacklisted, err := s.TokenStore.Exists(ctx, refreshTokenStr)
	if err != nil {
		return "", "", 0, err
	}
	if blacklisted {
		return "", "", 0, ErrTokenBlacklisted
	}

	// 2. 解析refresh token
	claims, err := s.TokenMgr.ParseRefresh(refreshTokenStr)
	if err != nil {
		return "", "", 0, err
	}

	// 3. 将旧refresh token加入黑名单（轮转模式）
	_ = s.TokenStore.Add(ctx, refreshTokenStr, claims.ExpiresAt.Unix())

	// 4. 签发新access token
	newAccessToken, expiresAt, err := s.TokenMgr.Generate(claims.UserID, claims.Username)
	if err != nil {
		return "", "", 0, err
	}

	// 5. 签发新refresh token
	newRefreshToken, _, err := s.TokenMgr.GenerateRefresh(claims.UserID, claims.Username)
	if err != nil {
		return "", "", 0, err
	}

	return newAccessToken, newRefreshToken, expiresAt, nil
}
