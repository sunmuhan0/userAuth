package model

import "time"

// Config 配置项模型
type Config struct {
	ID        int64     `json:"id" db:"id"`
	Service   string    `json:"service" db:"service"`
	Key       string    `json:"key" db:"key"`
	Value     string    `json:"value" db:"value"`
	Encrypted int       `json:"encrypted" db:"encrypted"` // 0=明文, 1=密文
	Version   int       `json:"version" db:"version"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
