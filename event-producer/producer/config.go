package producer

import "fmt"

// RMQConfig RocketMQ生产者配置
// 实现 inji.Startable，Start中填充配置值
// 当前写死，后期从配置中心获取
type RMQConfig struct {
	NameServer string // NameServer地址
	GroupName  string // 生产者组名
}

// Start 实现 inji.Startable 接口
func (c *RMQConfig) Start() error {
	c.NameServer = "127.0.0.1:9876"
	c.GroupName = "ttuser-producer-group"
	fmt.Println("[rmqProducerConfig] initialized")
	return nil
}
