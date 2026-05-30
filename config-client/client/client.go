package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Config 客户端配置
type Config struct {
	Endpoint    string // config-server 地址，默认 http://127.0.0.1:7963
	Env         string // 当前环境，如 prod/staging/preview
	ServiceName string // 当前服务名，如 auth-server
	Token       string // 认证 token
	ConfigDir   string // 配置文件本地目录，默认 ./config
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Endpoint:  "http://127.0.0.1:7963",
		Token:     "ttuser-config-token-2024",
		ConfigDir: "./config",
	}
}

// ConfigClient 配置客户端
type ConfigClient struct {
	endpoint    string
	env         string
	serviceName string
	configDir   string
	token       string
	httpClient  *http.Client
}

// New 创建配置客户端
func New(cfg *Config) *ConfigClient {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = DefaultConfig().Endpoint
	}
	token := cfg.Token
	if token == "" {
		token = DefaultConfig().Token
	}
	configDir := cfg.ConfigDir
	if configDir == "" {
		configDir = DefaultConfig().ConfigDir
	}
	return &ConfigClient{
		endpoint:    endpoint,
		env:         cfg.Env,
		serviceName: cfg.ServiceName,
		configDir:   configDir,
		token:       token,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

// FetchConfigs 从 config-server 下载所有配置文件到 {configDir}/{serviceName}/
func (c *ConfigClient) FetchConfigs() error {
	if c.env == "" || c.serviceName == "" {
		return fmt.Errorf("env and serviceName are required")
	}

	u := fmt.Sprintf("%s/config/files?env=%s&service=%s", c.endpoint, c.env, c.serviceName)
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request config-server failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("config-server returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Code int `json:"code"`
		Data struct {
			Files []struct {
				Name    string `json:"name"`
				Content string `json:"content"`
			} `json:"files"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("unmarshal response failed: %w", err)
	}

	if result.Code != 0 {
		return fmt.Errorf("config-server error: code=%d", result.Code)
	}

	localDir := filepath.Join(c.configDir, c.serviceName)
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return fmt.Errorf("create local dir %s failed: %w", localDir, err)
	}

	for _, f := range result.Data.Files {
		fp := filepath.Join(localDir, f.Name)
		if err := os.WriteFile(fp, []byte(f.Content), 0644); err != nil {
			return fmt.Errorf("write file %s failed: %w", f.Name, err)
		}
	}

	fmt.Printf("[config-client] downloaded %d config files to %s\n", len(result.Data.Files), localDir)
	return nil
}

// LoadFile 读取配置文件并反序列化
// target 必须是指针类型
func LoadFile(serviceName, filename string, target interface{}) error {
	// 路径 ./config
	fp := filepath.Join(DefaultConfig().ConfigDir, serviceName, filename)
	data, err := os.ReadFile(fp)
	if err == nil {
		if err := json.Unmarshal(data, target); err == nil {
			return nil
		}
	}

	return fmt.Errorf("read config file %s/%s failed", serviceName, filename)
}
