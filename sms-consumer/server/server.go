package server

import (
	"fmt"
	"log"

	"github.com/streadway/amqp"

	"ttuser/sms-consumer/internal/handler"
)

// RMQConfig SMS消费者RMQ配置
// 实现 inji.Startable，Start中填充配置值
// 当前写死，后期从配置中心获取
type RMQConfig struct {
	URL          string
	Exchange     string
	ExchangeType string
	RoutingKey   string
	Queue        string
	PrefetchCnt  int
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
// 聚合RMQ配置、事件处理器，通过inji注入
type SMSConsumerServer struct {
	Config  *RMQConfig          `inject:"rmqConfig"`
	Handler *handler.SMSHandler `inject:"smsHandler"`
	conn    *amqp.Connection
	ch      *amqp.Channel
	done    chan struct{}
}

// Start 实现 inji.Startable 接口，启动RMQ消费
func (s *SMSConsumerServer) Start() error {
	s.done = make(chan struct{})

	var err error
	s.conn, err = amqp.Dial(s.Config.URL)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	s.ch, err = s.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}

	// 声明交换机
	err = s.ch.ExchangeDeclare(
		s.Config.Exchange,
		s.Config.ExchangeType,
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	// 声明队列
	_, err = s.ch.QueueDeclare(
		s.Config.Queue,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// 绑定队列到交换机
	err = s.ch.QueueBind(
		s.Config.Queue,
		s.Config.RoutingKey,
		s.Config.Exchange,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	// 设置QoS
	if s.Config.PrefetchCnt > 0 {
		err = s.ch.Qos(s.Config.PrefetchCnt, 0, false)
		if err != nil {
			return fmt.Errorf("failed to set QoS: %w", err)
		}
	}

	// 开始消费
	msgs, err := s.ch.Consume(
		s.Config.Queue,
		"",    // consumer tag
		false, // auto-ack（手动确认）
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	log.Printf("[sms-consumer] started, queue=%s, routingKey=%s", s.Config.Queue, s.Config.RoutingKey)

	go s.consume(msgs)

	return nil
}

// consume 消费消息循环
func (s *SMSConsumerServer) consume(msgs <-chan amqp.Delivery) {
	for {
		select {
		case <-s.done:
			return
		case msg, ok := <-msgs:
			if !ok {
				log.Println("[sms-consumer] channel closed, exiting")
				return
			}
			s.processMessage(msg)
		}
	}
}

// processMessage 处理单条消息
func (s *SMSConsumerServer) processMessage(msg amqp.Delivery) {
	if err := s.Handler.Handle(msg.Body); err != nil {
		log.Printf("[sms-consumer] handler error: %v, body: %s", err, string(msg.Body))
		msg.Nack(false, true)
		return
	}
	msg.Ack(false)
}

// Close 实现 inji.Closeable 接口
func (s *SMSConsumerServer) Close() {
	if s.done != nil {
		close(s.done)
	}
	if s.ch != nil {
		s.ch.Close()
	}
	if s.conn != nil {
		s.conn.Close()
	}
	fmt.Println("[sms-consumer-server] closed")
}
