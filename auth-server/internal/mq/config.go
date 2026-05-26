package mq

import (
	"fmt"
	"reflect"

	"github.com/teou/implmap"
)

func init() {
	// 注册 IRMQProducerConfig 接口的实现
	// inji通过implmap将 inject:"rmqProducerConfig" 映射到该类型
	implmap.Add("rmqProducerConfig", reflect.TypeOf((*ProducerConfig)(nil)))
}

// ProducerConfig auth-server的RMQ生产者配置
// 实现 producer.IRMQProducerConfig 接口
// 当前写死，后期从配置中心获取
type ProducerConfig struct {
	url          string
	exchange     string
	exchangeType string
}

// Start 实现 inji.Startable 接口
func (c *ProducerConfig) Start() error {
	c.url = "amqp://guest:guest@127.0.0.1:5672/"
	c.exchange = "user.events"
	c.exchangeType = "topic"
	fmt.Println("[auth-server rmqProducerConfig] initialized")
	return nil
}

func (c *ProducerConfig) GetURL() string          { return c.url }
func (c *ProducerConfig) GetExchange() string     { return c.exchange }
func (c *ProducerConfig) GetExchangeType() string { return c.exchangeType }
