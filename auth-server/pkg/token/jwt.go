package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

var (
	ErrTokenInvalid      = errors.New("token is invalid")
	ErrTokenExpired      = errors.New("token has expired")
	ErrTokenTypeMismatch = errors.New("token type mismatch")
)

// 默认配置，后续从配置中心获取
const (
	defaultSecret        = "my-secret-key-for-ttuser-2024"
	defaultAccessExpire  = 2 * time.Hour
	defaultRefreshExpire = 7 * 24 * time.Hour
)

// Claims 自定义JWT Claims
type Claims struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

// JWTManager JWT实现的token管理器
type JWTManager struct {
	Secret            string
	AccessExpireTime  time.Duration
	RefreshExpireTime time.Duration
}

// Start 实现 inji.Startable 接口，支持自动注册
func (m *JWTManager) Start() error {
	m.init()
	return nil
}

func (m *JWTManager) init() {
	m.Secret = defaultSecret // TODO: 后续从配置中心获取
	m.AccessExpireTime = defaultAccessExpire
	m.RefreshExpireTime = defaultRefreshExpire
}

// Generate 生成access token
func (m *JWTManager) Generate(userID, username string) (string, int64, error) {
	return m.generateToken(userID, username, TokenTypeAccess, m.AccessExpireTime)
}

// GenerateRefresh 生成refresh token
func (m *JWTManager) GenerateRefresh(userID, username string) (string, int64, error) {
	return m.generateToken(userID, username, TokenTypeRefresh, m.RefreshExpireTime)
}

// Parse 解析并验证access token
func (m *JWTManager) Parse(tokenStr string) (*Claims, error) {
	claims, err := m.parseToken(tokenStr)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != TokenTypeAccess {
		return nil, ErrTokenTypeMismatch
	}
	return claims, nil
}

// ParseRefresh 解析并验证refresh token
func (m *JWTManager) ParseRefresh(tokenStr string) (*Claims, error) {
	claims, err := m.parseToken(tokenStr)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != TokenTypeRefresh {
		return nil, ErrTokenTypeMismatch
	}
	return claims, nil
}

func (m *JWTManager) generateToken(userID, username, tokenType string, expire time.Duration) (string, int64, error) {
	expiresAt := time.Now().Add(expire)
	claims := &Claims{
		UserID:    userID,
		Username:  username,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "ttuser",
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := t.SignedString([]byte(m.Secret))
	if err != nil {
		return "", 0, err
	}
	return tokenStr, expiresAt.Unix(), nil
}

func (m *JWTManager) parseToken(tokenStr string) (*Claims, error) {
	t, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return []byte(m.Secret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}
	claims, ok := t.Claims.(*Claims)
	if !ok || !t.Valid {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}
