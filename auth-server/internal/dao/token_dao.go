package dao

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"ttuser/data-store/engine"
)

// TokenRecord token黑名单记录
type TokenRecord struct {
	TokenHash string    `db:"token_hash"`
	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
}

// TokenDAO token黑名单数据访问对象
type TokenDAO struct {
	Mysql engine.IMysqlClient `inject:"procMysqlClient"`
}

// Add 将token加入黑名单（存SHA256 hash）
func (d *TokenDAO) Add(ctx context.Context, token string, expiresAt int64) error {
	record := &TokenRecord{
		TokenHash: hashToken(token),
		ExpiresAt: time.Unix(expiresAt, 0),
		CreatedAt: time.Now(),
	}

	// INSERT IGNORE 避免重复插入
	_, err := d.Mysql.ExecContext(ctx,
		"INSERT IGNORE INTO token_blacklist (token_hash, expires_at, created_at) VALUES (?, ?, ?)",
		record.TokenHash, record.ExpiresAt, record.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to add token to blacklist: %w", err)
	}
	return nil
}

// Exists 检查token是否在黑名单中
func (d *TokenDAO) Exists(ctx context.Context, token string) (bool, error) {
	hash := hashToken(token)
	count, err := d.Mysql.GetCount(
		"SELECT COUNT(1) FROM token_blacklist WHERE token_hash = ?", hash,
	)
	if err != nil {
		return false, fmt.Errorf("failed to check token blacklist: %w", err)
	}
	return count > 0, nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
