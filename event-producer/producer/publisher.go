package producer

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/streadway/amqp"
)

// IEventPublisher 事件发布接口
// 所有下游服务（SMS、邮件、推送等）通过该接口发布事件到MQ
type IEventPublisher interface {
	// Publish 发布事件，routingKey决定哪些消费者收到消息
	Publish(routingKey string, event *Event) error
	// Close 关闭连接
	Close()
}

// RMQPublisher 基于RabbitMQ的事件发布实现
type RMQPublisher struct {
	config *RMQConfig
	conn   *amqp.Connection
	ch     *amqp.Channel
	mu     sync.Mutex
}

// NewRMQPublisher 创建RMQ事件发布者
func NewRMQPublisher(config *RMQConfig) *RMQPublisher {
	return &RMQPublisher{
		config: config,
	}
}

// Connect 建立RabbitMQ连接并声明交换机
func (p *RMQPublisher) Connect() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var err error
	p.conn, err = amqp.Dial(p.config.URL)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	p.ch, err = p.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}

	// 声明交换机（topic类型，支持通配符路由）
	err = p.ch.ExchangeDeclare(
		p.config.Exchange,     // name
		p.config.ExchangeType, // type
		true,                  // durable
		false,                 // auto-deleted
		false,                 // internal
		false,                 // no-wait
		nil,                   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	return nil
}

// Publish 发布事件到MQ
func (p *RMQPublisher) Publish(routingKey string, event *Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	err = p.ch.Publish(
		p.config.Exchange, // exchange
		routingKey,        // routing key
		false,             // mandatory
		false,             // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish event [%s]: %w", routingKey, err)
	}

	fmt.Printf("[event-producer] published event: type=%s, routingKey=%s\n", event.Type, routingKey)
	return nil
}

// Close 关闭连接
func (p *RMQPublisher) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.ch != nil {
		p.ch.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
}
