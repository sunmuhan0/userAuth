package producer

import (
	"fmt"

	"github.com/teou/inji"

	configclient "ttuser/config-client/client"
)

// RMQConfig RocketMQ生产者配置
// Start时从配置中心获取
type RMQConfig struct {
	NameServer string
	GroupName  string
}

// Start 实现 inji.Startable 接口
func (c *RMQConfig) Start() error {
	var rmqConf struct {
		NameServer string `json:"name_server"`
		GroupName  string `json:"group_name"`
	}
	svc := "event-producer"
	if v, ok := inji.Find("serverName"); ok {
		svc = v.(string)
	}
	if err := configclient.LoadFile(svc, "rocketmq.json", &rmqConf); err != nil {
		return fmt.Errorf("[rmqProducerConfig] load rocketmq config failed: %w", err)
	}
	c.NameServer = rmqConf.NameServer
	c.GroupName = rmqConf.GroupName

	fmt.Printf("[rmqProducerConfig] initialized: nameServer=%s, group=%s\n", c.NameServer, c.GroupName)
	return nil
}
