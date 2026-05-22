package service

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"ttuser/auth-server/internal/dao"
	"ttuser/auth-server/internal/model"
	"ttuser/auth-server/pkg/token"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("username already exists")
	ErrTokenBlacklisted   = errors.New("token has been revoked")
	ErrPasswordTooShort   = errors.New("password must be at least 6 characters")
	ErrPasswordTooLong    = errors.New("password must not exceed 72 characters")
	ErrUsernameTooLong    = errors.New("username must not exceed 64 characters")
)

// AuthServiceImpl 认证服务实现
type AuthServiceImpl struct {
	UserDAO  *dao.UserDAO      `inject:"userDAO"`
	TokenDAO *dao.TokenDAO     `inject:"tokenDAO"`
	TokenMgr *token.JWTManager `inject:"tokenManager"`
}

// Register 用户注册
func (s *AuthServiceImpl) Register(ctx context.Context, username, password, nickname, email string) (*model.User, error) {
	// 输入校验
	if len(username) > 64 {
		return nil, ErrUsernameTooLong
	}
	if len(password) < 6 {
		return nil, ErrPasswordTooShort
	}
	if len(password) > 72 {
		return nil, ErrPasswordTooLong
	}

	// bcrypt 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	record, err := s.UserDAO.Create(ctx, username, string(hashedPassword), nickname, email)
	if err != nil {
		if errors.Is(err, dao.ErrUserAlreadyExists) {
			return nil, ErrUserAlreadyExists
		}
		return nil, err
	}

	return recordToUser(record), nil
}

// Login 用户登录
func (s *AuthServiceImpl) Login(ctx context.Context, username, password string) (string, string, int64, *model.User, error) {
	record, err := s.UserDAO.GetByUsername(ctx, username)
	if err != nil {
		return "", "", 0, nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(record.Password), []byte(password)); err != nil {
		return "", "", 0, nil, ErrInvalidCredentials
	}

	accessToken, expiresAt, err := s.TokenMgr.Generate(record.ID, record.Username)
	if err != nil {
		return "", "", 0, nil, err
	}

	refreshToken, _, err := s.TokenMgr.GenerateRefresh(record.ID, record.Username)
	if err != nil {
		return "", "", 0, nil, err
	}

	return accessToken, refreshToken, expiresAt, recordToUser(record), nil
}

// Logout 注销
func (s *AuthServiceImpl) Logout(ctx context.Context, accessToken, refreshToken string) error {
	if accessToken != "" {
		claims, err := s.TokenMgr.Parse(accessToken)
		if err == nil {
			if err := s.TokenDAO.Add(ctx, accessToken, claims.ExpiresAt.Unix()); err != nil {
				return fmt.Errorf("failed to blacklist access token: %w", err)
			}
		}
	}
	if refreshToken != "" {
		claims, err := s.TokenMgr.ParseRefresh(refreshToken)
		if err == nil {
			if err := s.TokenDAO.Add(ctx, refreshToken, claims.ExpiresAt.Unix()); err != nil {
				return fmt.Errorf("failed to blacklist refresh token: %w", err)
			}
		}
	}
	return nil
}

// GetUserInfo 获取用户信息
func (s *AuthServiceImpl) GetUserInfo(ctx context.Context, userID string) (*model.User, error) {
	record, err := s.UserDAO.GetByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return recordToUser(record), nil
}

// UpdateUserInfo 更新用户信息
func (s *AuthServiceImpl) UpdateUserInfo(ctx context.Context, userID, nickname, email, avatar string) (*model.User, error) {
	record, err := s.UserDAO.Update(ctx, userID, nickname, email, avatar)
	if err != nil {
		return nil, err
	}
	return recordToUser(record), nil
}

// ValidateToken 验证token
func (s *AuthServiceImpl) ValidateToken(ctx context.Context, tokenStr string) (*token.Claims, error) {
	blacklisted, err := s.TokenDAO.Exists(ctx, tokenStr)
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

// RefreshToken 续签token
func (s *AuthServiceImpl) RefreshToken(ctx context.Context, refreshTokenStr string) (string, string, int64, error) {
	blacklisted, err := s.TokenDAO.Exists(ctx, refreshTokenStr)
	if err != nil {
		return "", "", 0, err
	}
	if blacklisted {
		return "", "", 0, ErrTokenBlacklisted
	}

	claims, err := s.TokenMgr.ParseRefresh(refreshTokenStr)
	if err != nil {
		return "", "", 0, err
	}

	// 旧refresh token加入黑名单（轮转）
	if err := s.TokenDAO.Add(ctx, refreshTokenStr, claims.ExpiresAt.Unix()); err != nil {
		return "", "", 0, fmt.Errorf("failed to blacklist old refresh token: %w", err)
	}

	newAccessToken, expiresAt, err := s.TokenMgr.Generate(claims.UserID, claims.Username)
	if err != nil {
		return "", "", 0, err
	}

	newRefreshToken, _, err := s.TokenMgr.GenerateRefresh(claims.UserID, claims.Username)
	if err != nil {
		return "", "", 0, err
	}

	return newAccessToken, newRefreshToken, expiresAt, nil
}

func recordToUser(r *dao.UserRecord) *model.User {
	return &model.User{
		ID:        r.ID,
		Username:  r.Username,
		Nickname:  r.Nickname,
		Email:     r.Email,
		Avatar:    r.Avatar,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}
