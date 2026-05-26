package producer

import (
	"fmt"

	"ttuser/config-client/client"
)

// RMQConfig RocketMQ生产者配置
// Start时从配置中心获取，获取失败则用默认值
type RMQConfig struct {
	NameServer string // NameServer地址
	GroupName  string // 生产者组名
}

// Start 实现 inji.Startable 接口
func (c *RMQConfig) Start() error {
	// 从配置中心获取
	cfg := client.DefaultConfig()
	cfg.ServiceName = "event-producer"
	configClient := client.New(cfg)
	if err := configClient.Start(0); err != nil {
		fmt.Printf("[rmqProducerConfig] config-center unavailable, using defaults: %v\n", err)
		c.NameServer = "127.0.0.1:9876"
		c.GroupName = "ttuser-producer-group"
	} else {
		var rmqConf struct {
			NameServer string `json:"name_server"`
			GroupName  string `json:"group_name"`
		}
		if err := configClient.Get("rocketmq", &rmqConf); err != nil {
			fmt.Printf("[rmqProducerConfig] config key 'rocketmq' not found, using defaults: %v\n", err)
			c.NameServer = "127.0.0.1:9876"
			c.GroupName = "ttuser-producer-group"
		} else {
			c.NameServer = rmqConf.NameServer
			c.GroupName = rmqConf.GroupName
		}
	}

	fmt.Printf("[rmqProducerConfig] initialized: nameServer=%s, group=%s\n", c.NameServer, c.GroupName)
	return nil
}
