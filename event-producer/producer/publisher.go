package producer

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/streadway/amqp"
)

// IEventPublisher 事件发布接口
// 任何服务需要发MQ消息，注入该接口即可
type IEventPublisher interface {
	Publish(routingKey string, event *Event) error
}

// RMQPublisher 基于RabbitMQ的事件发布实现
// 通过inji注入Config，Start()时自动连接RabbitMQ
// 任何服务引用event-producer模块，注入*RMQPublisher即可使用
type RMQPublisher struct {
	Config *RMQConfig `inject:"rmqProducerConfig"`
	conn   *amqp.Connection
	ch     *amqp.Channel
	mu     sync.Mutex
}

// Start 实现 inji.Startable 接口，建立连接并声明交换机
func (p *RMQPublisher) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var err error
	p.conn, err = amqp.Dial(p.Config.URL)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	p.ch, err = p.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}

	// 声明交换机
	err = p.ch.ExchangeDeclare(
		p.Config.Exchange,
		p.Config.ExchangeType,
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	fmt.Printf("[event-producer] connected to RabbitMQ, exchange=%s\n", p.Config.Exchange)
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
		p.Config.Exchange, // exchange
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

// Close 实现 inji.Closeable 接口
func (p *RMQPublisher) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.ch != nil {
		p.ch.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
	fmt.Println("[event-producer] connection closed")
}
