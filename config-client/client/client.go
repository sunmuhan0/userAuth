package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"ttuser/pkg/crypto"
)

// ConfigClient 配置中心客户端
// 启动时拉取配置，支持定时刷新
type ConfigClient struct {
	endpoint    string // config-server 地址
	serviceName string // 本服务名
	token       string // 认证token
	cache       map[string]string // key → 解密后的value
	mu          sync.RWMutex
	httpClient  *http.Client
	stopCh      chan struct{}
}

// Config 客户端配置
type Config struct {
	Endpoint        string        // config-server 地址，如 http://127.0.0.1:7963
	ServiceName     string        // 本服务名，如 auth-server
	Token           string        // 认证token
	RefreshInterval time.Duration // 定时刷新间隔，0表示不刷新
}

// DefaultConfig 默认配置（写死，后期可从启动参数获取）
func DefaultConfig() *Config {
	return &Config{
		Endpoint:        "http://127.0.0.1:7963",
		ServiceName:     "",
		Token:           "ttuser-config-token-2024",
		RefreshInterval: 60 * time.Second,
	}
}

// configResponse config-server 返回结构
type configResponse struct {
	Code int `json:"code"`
	Data *struct {
		Value     string `json:"value"`
		Encrypted int    `json:"encrypted"`
	} `json:"data"`
	Message string `json:"message"`
}

// configsResponse config-server 批量返回
type configsResponse struct {
	Code int `json:"code"`
	Data []struct {
		Key       string `json:"key"`
		Value     string `json:"value"`
		Encrypted int    `json:"encrypted"`
	} `json:"data"`
}

// New 创建配置客户端
func New(cfg *Config) *ConfigClient {
	c := &ConfigClient{
		endpoint:    cfg.Endpoint,
		serviceName: cfg.ServiceName,
		token:       cfg.Token,
		cache:       make(map[string]string),
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		stopCh:      make(chan struct{}),
	}
	return c
}

// Start 启动客户端：拉取所有配置 + 启动定时刷新
func (c *ConfigClient) Start(refreshInterval time.Duration) error {
	if err := c.refresh(); err != nil {
		return fmt.Errorf("initial config fetch failed: %w", err)
	}
	if refreshInterval > 0 {
		go c.refreshLoop(refreshInterval)
	}
	fmt.Printf("[config-client] started, service=%s, endpoint=%s\n", c.serviceName, c.endpoint)
	return nil
}

// Stop 停止定时刷新
func (c *ConfigClient) Stop() {
	close(c.stopCh)
}

// Get 获取配置值并反序列化到target
// target 必须是指针类型
func (c *ConfigClient) Get(key string, target interface{}) error {
	c.mu.RLock()
	value, ok := c.cache[key]
	c.mu.RUnlock()

	if !ok {
		return fmt.Errorf("config not found: service=%s, key=%s", c.serviceName, key)
	}

	if err := json.Unmarshal([]byte(value), target); err != nil {
		return fmt.Errorf("unmarshal config [%s] failed: %w", key, err)
	}
	return nil
}

// GetRaw 获取配置原始字符串值
func (c *ConfigClient) GetRaw(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.cache[key]
	return v, ok
}

// refresh 从config-server拉取所有配置并更新缓存
func (c *ConfigClient) refresh() error {
	u := fmt.Sprintf("%s/configs?service=%s", c.endpoint, url.QueryEscape(c.serviceName))
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

	var result configsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("unmarshal response failed: %w", err)
	}

	if result.Code != 0 {
		return fmt.Errorf("config-server error: code=%d", result.Code)
	}

	// 更新缓存
	newCache := make(map[string]string, len(result.Data))
	for _, item := range result.Data {
		value := item.Value
		// 如果标记为加密，自动解密
		if item.Encrypted == 1 {
			decrypted, err := crypto.Decrypt(value)
			if err != nil {
				fmt.Printf("[config-client] decrypt config [%s] failed: %v, skip\n", item.Key, err)
				continue
			}
			value = decrypted
		}
		newCache[item.Key] = value
	}

	c.mu.Lock()
	c.cache = newCache
	c.mu.Unlock()

	return nil
}

// refreshLoop 定时刷新
func (c *ConfigClient) refreshLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := c.refresh(); err != nil {
				fmt.Printf("[config-client] refresh failed: %v\n", err)
			}
		case <-c.stopCh:
			return
		}
	}
}
