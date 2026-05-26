package service

import (
	"ttuser/config-server/internal/dao"
	"ttuser/config-server/internal/model"
	"ttuser/pkg/crypto"
)

// ConfigService 配置管理服务
type ConfigService struct {
	ConfigDAO *dao.ConfigDAO `inject:"configDAO"`
}

// Get 获取单个配置
func (s *ConfigService) Get(service, key string) (*model.Config, error) {
	return s.ConfigDAO.Get(service, key)
}

// List 获取服务的所有配置
func (s *ConfigService) List(service string) ([]*model.Config, error) {
	return s.ConfigDAO.List(service)
}

// Set 创建或更新配置（明文存储）
func (s *ConfigService) Set(service, key, value string) error {
	return s.ConfigDAO.Set(service, key, value, 0)
}

// SetEncrypted 创建或更新配置（加密存储）
// 对value进行AES加密后存入数据库
func (s *ConfigService) SetEncrypted(service, key, value string) error {
	encrypted, err := crypto.Encrypt(value)
	if err != nil {
		return err
	}
	return s.ConfigDAO.Set(service, key, encrypted, 1)
}
