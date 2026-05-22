package dao

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"ttuser/data-store/engine"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("username already exists")
)

// UserRecord 用户数据库记录
type UserRecord struct {
	ID        string    `db:"id"`
	Username  string    `db:"username"`
	Password  string    `db:"password"`
	Nickname  string    `db:"nickname"`
	Email     string    `db:"email"`
	Avatar    string    `db:"avatar"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// UserDAO 用户数据访问对象
type UserDAO struct {
	Mysql engine.IMysqlClient `inject:"procMysqlClient"`
}

// Create 创建用户
func (d *UserDAO) Create(ctx context.Context, username, password, nickname, email string) (*UserRecord, error) {
	id := uuid.New().String()
	now := time.Now()

	record := &UserRecord{
		ID:        id,
		Username:  username,
		Password:  password,
		Nickname:  nickname,
		Email:     email,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := d.Mysql.Add("users", record, false)
	if err != nil {
		if isDuplicateEntry(err) {
			return nil, ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return record, nil
}

// GetByID 根据ID获取用户
func (d *UserDAO) GetByID(ctx context.Context, id string) (*UserRecord, error) {
	result, err := d.Mysql.Query(
		(*UserRecord)(nil),
		"SELECT id, username, password, nickname, email, avatar, created_at, updated_at FROM users WHERE id = ?",
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}
	if result == nil {
		return nil, ErrUserNotFound
	}
	return result.(*UserRecord), nil
}

// GetByUsername 根据用户名获取用户
func (d *UserDAO) GetByUsername(ctx context.Context, username string) (*UserRecord, error) {
	result, err := d.Mysql.Query(
		(*UserRecord)(nil),
		"SELECT id, username, password, nickname, email, avatar, created_at, updated_at FROM users WHERE username = ?",
		username,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}
	if result == nil {
		return nil, ErrUserNotFound
	}
	return result.(*UserRecord), nil
}

// Update 更新用户信息
func (d *UserDAO) Update(ctx context.Context, id, nickname, email, avatar string) (*UserRecord, error) {
	fieldsToUpdate := make([]string, 0)
	record := &UserRecord{ID: id}

	if nickname != "" {
		record.Nickname = nickname
		fieldsToUpdate = append(fieldsToUpdate, "nickname")
	}
	if email != "" {
		record.Email = email
		fieldsToUpdate = append(fieldsToUpdate, "email")
	}
	if avatar != "" {
		record.Avatar = avatar
		fieldsToUpdate = append(fieldsToUpdate, "avatar")
	}

	if len(fieldsToUpdate) == 0 {
		return d.GetByID(ctx, id)
	}

	err := d.Mysql.Update("users", record, map[string]interface{}{"id": id}, fieldsToUpdate)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return d.GetByID(ctx, id)
}

// Delete 删除用户
func (d *UserDAO) Delete(ctx context.Context, id string) error {
	rows, err := d.Mysql.Delete("users", map[string]interface{}{"id": id})
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

func isDuplicateEntry(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	for i := 0; i <= len(s)-4; i++ {
		if s[i:i+4] == "1062" {
			return true
		}
	}
	return false
}
