package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ConfigFile 配置文件信息
type ConfigFile struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// ConfigService 配置管理服务，从文件系统读取配置
type ConfigService struct {
	ConfigDir string // 配置根目录，如 ./config-center
}

// NewConfigService 创建配置服务
func NewConfigService(configDir string) *ConfigService {
	return &ConfigService{ConfigDir: configDir}
}

// ListFiles 获取合并后的配置文件列表
// 先加载 base/{service}/，再加载 {env}/{service}/，同名文件 env 覆盖 base
func (s *ConfigService) ListFiles(env, service string) ([]ConfigFile, error) {
	files := make(map[string]string)

	// 1. 加载 base/{service}/
	baseDir := filepath.Join(s.ConfigDir, "base", service)
	if entries, err := os.ReadDir(baseDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
				data, err := os.ReadFile(filepath.Join(baseDir, e.Name()))
				if err != nil {
					return nil, fmt.Errorf("read base config file %s failed: %w", e.Name(), err)
				}
				files[e.Name()] = string(data)
			}
		}
	}

	// 2. 加载 {env}/{service}/，覆盖同名文件
	envDir := filepath.Join(s.ConfigDir, env, service)
	if entries, err := os.ReadDir(envDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
				data, err := os.ReadFile(filepath.Join(envDir, e.Name()))
				if err != nil {
					return nil, fmt.Errorf("read env config file %s failed: %w", e.Name(), err)
				}
				files[e.Name()] = string(data)
			}
		}
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no config files found for env=%s, service=%s", env, service)
	}

	result := make([]ConfigFile, 0, len(files))
	for name, content := range files {
		result = append(result, ConfigFile{Name: name, Content: content})
	}
	return result, nil
}
