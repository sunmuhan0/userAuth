package log

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// LokiConfig Loki推送配置
type LokiConfig struct {
	Enable   bool              // 是否启用Loki推送
	Endpoint string            // Loki push API地址，如 http://127.0.0.1:3100/loki/api/v1/push
	Labels   map[string]string // 固定标签，如 {"app":"auth-server","env":"dev"}
	BatchSize int              // 批量发送条数阈值
	FlushInterval time.Duration // 定时刷新间隔
}

// DefaultLokiConfig 默认Loki配置（写死，后期配置中心获取）
func DefaultLokiConfig() *LokiConfig {
	return &LokiConfig{
		Enable:        false, // 默认关闭，需要时手动开启
		Endpoint:      "http://127.0.0.1:3100/loki/api/v1/push",
		Labels:        map[string]string{"app": "ttuser", "env": "dev"},
		BatchSize:     100,
		FlushInterval: 3 * time.Second,
	}
}

// lokiClient Loki日志推送客户端
type lokiClient struct {
	config  *LokiConfig
	entries []lokiEntry
	mu      sync.Mutex
	client  *http.Client
	stopCh  chan struct{}
}

type lokiEntry struct {
	Timestamp time.Time
	Line      string
}

// lokiPushRequest Loki push API请求体
type lokiPushRequest struct {
	Streams []lokiStream `json:"streams"`
}

type lokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

var loki *lokiClient

// InitLoki 初始化Loki推送客户端
// 在应用启动时调用，如 log.InitLoki(log.DefaultLokiConfig())
func InitLoki(config *LokiConfig) {
	if config == nil || !config.Enable {
		return
	}
	loki = &lokiClient{
		config:  config,
		entries: make([]lokiEntry, 0, config.BatchSize),
		client:  &http.Client{Timeout: 5 * time.Second},
		stopCh:  make(chan struct{}),
	}
	go loki.flushLoop()
	fmt.Println("[log] loki push client initialized, endpoint:", config.Endpoint)
}

// pushToLoki 异步将日志条目加入批次
func pushToLoki(line string) {
	if loki == nil {
		return
	}
	loki.mu.Lock()
	loki.entries = append(loki.entries, lokiEntry{
		Timestamp: time.Now(),
		Line:      line,
	})
	shouldFlush := len(loki.entries) >= loki.config.BatchSize
	loki.mu.Unlock()

	if shouldFlush {
		go loki.flush()
	}
}

// flushLoop 定时刷新
func (l *lokiClient) flushLoop() {
	ticker := time.NewTicker(l.config.FlushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			l.flush()
		case <-l.stopCh:
			l.flush() // 停止前最后刷一次
			return
		}
	}
}

// flush 批量发送到Loki
func (l *lokiClient) flush() {
	l.mu.Lock()
	if len(l.entries) == 0 {
		l.mu.Unlock()
		return
	}
	entries := l.entries
	l.entries = make([]lokiEntry, 0, l.config.BatchSize)
	l.mu.Unlock()

	// 构建Loki push请求
	values := make([][]string, 0, len(entries))
	for _, e := range entries {
		// Loki要求时间戳为纳秒字符串
		ts := fmt.Sprintf("%d", e.Timestamp.UnixNano())
		values = append(values, []string{ts, e.Line})
	}

	req := lokiPushRequest{
		Streams: []lokiStream{
			{
				Stream: l.config.Labels,
				Values: values,
			},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		fmt.Printf("[log] loki marshal error: %v\n", err)
		return
	}

	resp, err := l.client.Post(l.config.Endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Printf("[log] loki push error: %v\n", err)
		return
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		fmt.Printf("[log] loki push unexpected status: %d\n", resp.StatusCode)
	}
}

// StopLoki 停止Loki客户端（程序退出前调用）
func StopLoki() {
	if loki != nil {
		close(loki.stopCh)
	}
}
