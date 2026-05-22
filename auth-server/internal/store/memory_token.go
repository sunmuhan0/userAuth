package store

import (
	"context"
	"sync"
	"time"
)

// MemoryTokenStore 内存实现的token黑名单存储
type MemoryTokenStore struct {
	mu     sync.RWMutex
	tokens map[string]int64
}

// Start 实现 inji.Startable 接口，支持自动注册
func (s *MemoryTokenStore) Start() error {
	s.init()
	return nil
}

func (s *MemoryTokenStore) init() {
	s.tokens = make(map[string]int64)
	go s.cleanup()
}

func (s *MemoryTokenStore) Add(_ context.Context, token string, expiresAt int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[token] = expiresAt
	return nil
}

func (s *MemoryTokenStore) Exists(_ context.Context, token string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.tokens[token]
	return exists, nil
}

func (s *MemoryTokenStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now().Unix()
		s.mu.Lock()
		for token, exp := range s.tokens {
			if exp < now {
				delete(s.tokens, token)
			}
		}
		s.mu.Unlock()
	}
}
