package producer

// RMQConfig RabbitMQ连接配置
// 当前先写死，后期从配置中心获取
type RMQConfig struct {
	URL          string // amqp://user:pass@host:port/vhost
	Exchange     string // 交换机名称
	ExchangeType string // 交换机类型 (direct/topic/fanout)
}

// DefaultConfig 默认配置（写死，后期从配置中心获取）
func DefaultConfig() *RMQConfig {
	return &RMQConfig{
		URL:          "amqp://guest:guest@127.0.0.1:5672/",
		Exchange:     "user.events",
		ExchangeType: "topic",
	}
}
