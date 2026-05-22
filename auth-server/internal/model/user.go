package model

import "time"

// User 用户模型
type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"-"`
	Nickname  string    `json:"nickname"`
	Email     string    `json:"email"`
	Avatar    string    `json:"avatar"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
