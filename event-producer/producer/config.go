package producer

import "fmt"

// RMQConfig RabbitMQ连接配置
// 实现 inji.Startable，Start中填充配置值
// 当前写死，后期从配置中心获取
type RMQConfig struct {
	URL          string // amqp://user:pass@host:port/vhost
	Exchange     string // 交换机名称
	ExchangeType string // 交换机类型 (direct/topic/fanout)
}

// Start 实现 inji.Startable 接口
func (c *RMQConfig) Start() error {
	c.URL = "amqp://guest:guest@127.0.0.1:5672/"
	c.Exchange = "user.events"
	c.ExchangeType = "topic"
	fmt.Println("[rmqProducerConfig] initialized")
	return nil
}
