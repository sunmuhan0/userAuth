package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
)

// 环境变量名：加密密钥（32字节=AES-256）
const envKey = "CONFIG_ENCRYPT_KEY"

// 默认密钥（仅开发环境使用，生产环境必须通过环境变量注入）
const fallbackKey = "ttuser-config-secret-key-2024!!"

// getKey 获取加密密钥：优先从环境变量，其次用默认值
func getKey() string {
	if key := os.Getenv(envKey); key != "" {
		return key
	}
	return fallbackKey
}

// Encrypt AES-256-GCM 加密，返回base64编码的密文
func Encrypt(plaintext string) (string, error) {
	return EncryptWithKey(plaintext, getKey())
}

// Decrypt AES-256-GCM 解密，输入base64编码的密文
func Decrypt(ciphertext string) (string, error) {
	return DecryptWithKey(ciphertext, getKey())
}

// EncryptWithKey 使用指定密钥加密
func EncryptWithKey(plaintext, key string) (string, error) {
	keyBytes := padKey(key)
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", fmt.Errorf("create cipher failed: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM failed: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce failed: %w", err)
	}

	sealed := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// DecryptWithKey 使用指定密钥解密
func DecryptWithKey(ciphertext, key string) (string, error) {
	keyBytes := padKey(key)
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("base64 decode failed: %w", err)
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", fmt.Errorf("create cipher failed: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM failed: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt failed: %w", err)
	}

	return string(plaintext), nil
}

// padKey 确保密钥为32字节（AES-256要求）
func padKey(key string) []byte {
	b := []byte(key)
	if len(b) >= 32 {
		return b[:32]
	}
	padded := make([]byte, 32)
	copy(padded, b)
	return padded
}
