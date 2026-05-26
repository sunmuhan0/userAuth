package dao

import (
	"fmt"

	"ttuser/config-server/internal/model"
	"ttuser/data-store/engine"
)

// ConfigDAO 配置数据访问对象
type ConfigDAO struct {
	Mysql engine.IMysqlClient `inject:"procMysqlClient"`
}

// Get 根据service和key获取配置
func (d *ConfigDAO) Get(service, key string) (*model.Config, error) {
	result, err := d.Mysql.Query(
		(*model.Config)(nil),
		"SELECT id, service, `key`, value, encrypted, version, updated_at FROM configs WHERE service = ? AND `key` = ?",
		service, key,
	)
	if err != nil {
		return nil, fmt.Errorf("query config failed: %w", err)
	}
	if result == nil {
		return nil, nil
	}
	return result.(*model.Config), nil
}

// List 获取某个服务的所有配置
func (d *ConfigDAO) List(service string) ([]*model.Config, error) {
	results, err := d.Mysql.QueryList(
		(*model.Config)(nil),
		"SELECT id, service, `key`, value, encrypted, version, updated_at FROM configs WHERE service = ?",
		service,
	)
	if err != nil {
		return nil, fmt.Errorf("query configs failed: %w", err)
	}
	configs := make([]*model.Config, 0, len(results))
	for _, r := range results {
		configs = append(configs, r.(*model.Config))
	}
	return configs, nil
}

// Set 创建或更新配置
func (d *ConfigDAO) Set(service, key, value string, encrypted int) error {
	_, err := d.Mysql.Execute(
		"INSERT INTO configs (service, `key`, value, encrypted, version) VALUES (?, ?, ?, ?, 1) ON DUPLICATE KEY UPDATE value = VALUES(value), encrypted = VALUES(encrypted), version = version + 1",
		service, key, value, encrypted,
	)
	if err != nil {
		return fmt.Errorf("set config failed: %w", err)
	}
	return nil
}
