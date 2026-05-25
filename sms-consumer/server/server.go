package server

import (
	"fmt"

	"ttuser/event-consumer/consumer"
	"ttuser/sms-consumer/internal/handler"
)

// RMQConfig SMS消费者RMQ配置
// 实现 inji.Startable，Start中填充配置值
// 当前写死，后期从配置中心获取
type RMQConfig struct {
	consumer.RMQConfig
}

// Start 实现 inji.Startable 接口
func (c *RMQConfig) Start() error {
	c.URL = "amqp://guest:guest@127.0.0.1:5672/"
	c.Exchange = "user.events"
	c.ExchangeType = "topic"
	c.RoutingKey = "user.registered"
	c.Queue = "sms.user.registered"
	c.PrefetchCnt = 10
	fmt.Println("[rmqConfig] initialized")
	return nil
}

// SMSConsumerServer 短信消费服务
// 聚合RMQ配置、事件处理器、RMQ消费者，通过inji注入
type SMSConsumerServer struct {
	Config   *RMQConfig           `inject:"rmqConfig"`
	Handler  *handler.SMSHandler  `inject:"smsHandler"`
	consumer *consumer.RMQConsumer
}

// Start 实现 inji.Startable 接口，启动RMQ消费
func (s *SMSConsumerServer) Start() error {
	s.consumer = &consumer.RMQConsumer{}
	if err := s.consumer.Start(&s.Config.RMQConfig, s.Handler); err != nil {
		return fmt.Errorf("[sms-consumer-server] failed to start: %w", err)
	}
	fmt.Println("[sms-consumer-server] started")
	return nil
}

// Close 实现 inji.Closeable 接口
func (s *SMSConsumerServer) Close() {
	if s.consumer != nil {
		s.consumer.Close()
	}
	fmt.Println("[sms-consumer-server] closed")
}
