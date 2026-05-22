package token

import (
	"testing"
	"time"
)

func newTestManager() *JWTManager {
	return &JWTManager{
		Secret:            "test-secret",
		AccessExpireTime:  2 * time.Hour,
		RefreshExpireTime: 7 * 24 * time.Hour,
	}
}

func TestGenerate(t *testing.T) {
	mgr := newTestManager()

	tokenStr, expiresAt, err := mgr.Generate("user1", "admin")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if tokenStr == "" {
		t.Fatal("token should not be empty")
	}
	if expiresAt <= time.Now().Unix() {
		t.Fatal("expiresAt should be in the future")
	}
}

func TestGenerateRefresh(t *testing.T) {
	mgr := newTestManager()

	tokenStr, expiresAt, err := mgr.GenerateRefresh("user1", "admin")
	if err != nil {
		t.Fatalf("GenerateRefresh failed: %v", err)
	}
	if tokenStr == "" {
		t.Fatal("refresh token should not be empty")
	}
	// refresh token 过期时间应大于 access token
	accessToken, accessExp, _ := mgr.Generate("user1", "admin")
	_ = accessToken
	if expiresAt <= accessExp {
		t.Fatal("refresh token should expire later than access token")
	}
}

func TestParse(t *testing.T) {
	mgr := newTestManager()

	tokenStr, _, _ := mgr.Generate("user1", "admin")

	claims, err := mgr.Parse(tokenStr)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if claims.UserID != "user1" {
		t.Fatalf("expected user_id=user1, got %s", claims.UserID)
	}
	if claims.Username != "admin" {
		t.Fatalf("expected username=admin, got %s", claims.Username)
	}
	if claims.TokenType != TokenTypeAccess {
		t.Fatalf("expected token_type=access, got %s", claims.TokenType)
	}
}

func TestParseRefresh(t *testing.T) {
	mgr := newTestManager()

	tokenStr, _, _ := mgr.GenerateRefresh("user1", "admin")

	claims, err := mgr.ParseRefresh(tokenStr)
	if err != nil {
		t.Fatalf("ParseRefresh failed: %v", err)
	}
	if claims.UserID != "user1" {
		t.Fatalf("expected user_id=user1, got %s", claims.UserID)
	}
	if claims.TokenType != TokenTypeRefresh {
		t.Fatalf("expected token_type=refresh, got %s", claims.TokenType)
	}
}

func TestParseAccessTokenAsRefresh_ShouldFail(t *testing.T) {
	mgr := newTestManager()

	accessToken, _, _ := mgr.Generate("user1", "admin")

	_, err := mgr.ParseRefresh(accessToken)
	if err != ErrTokenTypeMismatch {
		t.Fatalf("expected ErrTokenTypeMismatch, got %v", err)
	}
}

func TestParseRefreshTokenAsAccess_ShouldFail(t *testing.T) {
	mgr := newTestManager()

	refreshToken, _, _ := mgr.GenerateRefresh("user1", "admin")

	_, err := mgr.Parse(refreshToken)
	if err != ErrTokenTypeMismatch {
		t.Fatalf("expected ErrTokenTypeMismatch, got %v", err)
	}
}

func TestParseExpiredToken(t *testing.T) {
	mgr := &JWTManager{
		Secret:            "test-secret",
		AccessExpireTime:  -1 * time.Hour, // 已过期
		RefreshExpireTime: 7 * 24 * time.Hour,
	}

	tokenStr, _, _ := mgr.Generate("user1", "admin")

	_, err := mgr.Parse(tokenStr)
	if err != ErrTokenExpired {
		t.Fatalf("expected ErrTokenExpired, got %v", err)
	}
}

func TestParseInvalidToken(t *testing.T) {
	mgr := newTestManager()

	_, err := mgr.Parse("invalid.token.string")
	if err != ErrTokenInvalid {
		t.Fatalf("expected ErrTokenInvalid, got %v", err)
	}
}

func TestParseWrongSecret(t *testing.T) {
	mgr1 := newTestManager()
	mgr2 := &JWTManager{
		Secret:            "different-secret",
		AccessExpireTime:  2 * time.Hour,
		RefreshExpireTime: 7 * 24 * time.Hour,
	}

	tokenStr, _, _ := mgr1.Generate("user1", "admin")

	_, err := mgr2.Parse(tokenStr)
	if err != ErrTokenInvalid {
		t.Fatalf("expected ErrTokenInvalid, got %v", err)
	}
}
