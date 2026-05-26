package server

import (
	"fmt"
	"log"

	"github.com/streadway/amqp"

	"ttuser/sms-consumer/internal/handler"
)

// TopicBinding 订阅组配置：一个routing key对应一个队列
type TopicBinding struct {
	RoutingKey string // topic，如 "user.registered"
	Queue      string // 队列名，如 "sms.user.registered"
}

// RMQConfig SMS消费者RMQ配置
// 支持多topic订阅
// 实现 inji.Startable，Start中填充配置值
// 当前写死，后期从配置中心获取
type RMQConfig struct {
	URL          string
	Exchange     string
	ExchangeType string
	PrefetchCnt  int
	Bindings     []TopicBinding // 多个订阅组
}

// Start 实现 inji.Startable 接口
func (c *RMQConfig) Start() error {
	c.URL = "amqp://guest:guest@127.0.0.1:5672/"
	c.Exchange = "user.events"
	c.ExchangeType = "topic"
	c.PrefetchCnt = 10
	c.Bindings = []TopicBinding{
		{RoutingKey: "user.registered", Queue: "sms.user.registered"},
		// 以后新增topic只需在这里加一行：
		// {RoutingKey: "order.created", Queue: "sms.order.created"},
	}
	fmt.Println("[rmqConfig] initialized")
	return nil
}

// SMSConsumerServer 短信消费服务
// 一个连接/channel消费多个队列，每个队列绑定不同topic
type SMSConsumerServer struct {
	Config  *RMQConfig          `inject:"rmqConfig"`
	Handler *handler.SMSHandler `inject:"smsHandler"`
	conn    *amqp.Connection
	ch      *amqp.Channel
	done    chan struct{}
}

// Start 实现 inji.Startable 接口
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

	// 设置QoS
	if s.Config.PrefetchCnt > 0 {
		err = s.ch.Qos(s.Config.PrefetchCnt, 0, false)
		if err != nil {
			return fmt.Errorf("failed to set QoS: %w", err)
		}
	}

	// 遍历所有订阅组，声明队列并绑定topic，启动消费
	for _, binding := range s.Config.Bindings {
		if err := s.startBinding(binding); err != nil {
			return err
		}
	}

	log.Printf("[sms-consumer] started, bindingCount=%d", len(s.Config.Bindings))
	return nil
}

// startBinding 为单个topic binding声明队列、绑定、启动消费goroutine
func (s *SMSConsumerServer) startBinding(binding TopicBinding) error {
	// 声明队列
	_, err := s.ch.QueueDeclare(
		binding.Queue,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue [%s]: %w", binding.Queue, err)
	}

	// 绑定队列到交换机
	err = s.ch.QueueBind(
		binding.Queue,
		binding.RoutingKey,
		s.Config.Exchange,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue [%s] to routingKey [%s]: %w", binding.Queue, binding.RoutingKey, err)
	}

	// 启动消费
	msgs, err := s.ch.Consume(
		binding.Queue,
		"",    // consumer tag
		false, // auto-ack（手动确认）
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to consume queue [%s]: %w", binding.Queue, err)
	}

	log.Printf("[sms-consumer] subscribed: queue=%s, routingKey=%s", binding.Queue, binding.RoutingKey)

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
