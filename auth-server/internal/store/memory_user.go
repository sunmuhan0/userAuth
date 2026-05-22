package store

import (
	"context"
	"errors"
	"sync"
	"time"

	"ttuser/auth-server/internal/model"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
)

// MemoryUserStore 内存实现的用户存储
type MemoryUserStore struct {
	mu     sync.RWMutex
	users  map[string]*model.User
	byName map[string]string
}

// Start 实现 inji.Startable 接口，支持自动注册
func (s *MemoryUserStore) Start() error {
	s.init()
	return nil
}

func (s *MemoryUserStore) init() {
	s.users = make(map[string]*model.User)
	s.byName = make(map[string]string)
	// 预置测试用户 admin/123456
	s.users["1"] = &model.User{
		ID:        "1",
		Username:  "admin",
		Password:  "$2a$10$kEMnguie9lFKZyfOnsZltuLMzjz1HdnUQDu1s/RO90GeZsKuaU64S", // 123456
		Nickname:  "管理员",
		Email:     "admin@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	s.byName["admin"] = "1"
}

func (s *MemoryUserStore) GetByID(_ context.Context, id string) (*model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *MemoryUserStore) GetByUsername(_ context.Context, username string) (*model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.byName[username]
	if !ok {
		return nil, ErrUserNotFound
	}
	return s.users[id], nil
}

func (s *MemoryUserStore) Create(_ context.Context, user *model.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.byName[user.Username]; exists {
		return ErrUserAlreadyExists
	}
	s.users[user.ID] = user
	s.byName[user.Username] = user.ID
	return nil
}

func (s *MemoryUserStore) Update(_ context.Context, user *model.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[user.ID]; !ok {
		return ErrUserNotFound
	}
	user.UpdatedAt = time.Now()
	s.users[user.ID] = user
	return nil
}

func (s *MemoryUserStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[id]
	if !ok {
		return ErrUserNotFound
	}
	delete(s.byName, user.Username)
	delete(s.users, id)
	return nil
}
