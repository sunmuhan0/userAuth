package consumer

import (
	"fmt"
	"log"

	"github.com/streadway/amqp"
)

// IEventHandler 事件处理接口
// 每个下游服务（短信、邮件、推送等）实现该接口
type IEventHandler interface {
	// Handle 处理一条原始消息，body为JSON字节
	Handle(body []byte) error
}

// HandlerFunc 函数适配器，方便用匿名函数实现IEventHandler
type HandlerFunc func(body []byte) error

func (f HandlerFunc) Handle(body []byte) error {
	return f(body)
}

// IEventConsumer 事件消费者接口
type IEventConsumer interface {
	Start() error
	Close()
}

// RMQConsumer 基于RabbitMQ的通用事件消费者
type RMQConsumer struct {
	conn *amqp.Connection
	ch   *amqp.Channel
	done chan struct{}
}

// Start 启动消费者
func (c *RMQConsumer) Start(config *RMQConfig, handler IEventHandler) error {
	c.done = make(chan struct{})

	var err error
	c.conn, err = amqp.Dial(config.URL)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	c.ch, err = c.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}

	// 声明交换机
	err = c.ch.ExchangeDeclare(
		config.Exchange,
		config.ExchangeType,
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
	_, err = c.ch.QueueDeclare(
		config.Queue,
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
	err = c.ch.QueueBind(
		config.Queue,
		config.RoutingKey,
		config.Exchange,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	// 设置QoS
	if config.PrefetchCnt > 0 {
		err = c.ch.Qos(config.PrefetchCnt, 0, false)
		if err != nil {
			return fmt.Errorf("failed to set QoS: %w", err)
		}
	}

	// 开始消费
	msgs, err := c.ch.Consume(
		config.Queue,
		"",    // consumer tag
		false, // auto-ack (手动确认)
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	log.Printf("[event-consumer] started, queue=%s, routingKey=%s", config.Queue, config.RoutingKey)

	go c.consume(msgs, handler)

	return nil
}

// consume 消费消息循环
func (c *RMQConsumer) consume(msgs <-chan amqp.Delivery, handler IEventHandler) {
	for {
		select {
		case <-c.done:
			return
		case msg, ok := <-msgs:
			if !ok {
				log.Println("[event-consumer] channel closed, exiting consumer loop")
				return
			}
			c.processMessage(msg, handler)
		}
	}
}

// processMessage 处理单条消息
func (c *RMQConsumer) processMessage(msg amqp.Delivery, handler IEventHandler) {
	if err := handler.Handle(msg.Body); err != nil {
		log.Printf("[event-consumer] handler error: %v, body: %s", err, string(msg.Body))
		// 处理失败，重新入队重试
		msg.Nack(false, true)
		return
	}
	// 处理成功，确认
	msg.Ack(false)
}

// Close 关闭连接
func (c *RMQConsumer) Close() {
	if c.done != nil {
		close(c.done)
	}
	if c.ch != nil {
		c.ch.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
	log.Println("[event-consumer] stopped")
}
