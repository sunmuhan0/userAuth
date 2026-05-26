package producer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	rmqproducer "github.com/apache/rocketmq-client-go/v2/producer"
)

// IEventPublisher 事件发布接口
// 任何服务需要发MQ消息，注入该接口即可
// topic: 消息主题，tag: 消息标签，payload: 任意struct（JSON序列化）
type IEventPublisher interface {
	Publish(topic string, tag string, payload interface{}) error
}

// RMQPublisher 基于RocketMQ的事件发布实现
// 通过inji注入Config，Start()时自动创建producer
// 业务方注入 *RMQPublisher 即可直接使用
type RMQPublisher struct {
	Config   *RMQConfig `inject:"rmqProducerConfig"`
	producer rocketmq.Producer
}

// Start 实现 inji.Startable 接口，创建并启动RocketMQ producer
func (p *RMQPublisher) Start() error {
	var err error
	p.producer, err = rocketmq.NewProducer(
		rmqproducer.WithNameServer([]string{p.Config.NameServer}),
		rmqproducer.WithGroupName(p.Config.GroupName),
		rmqproducer.WithRetry(3),
	)
	if err != nil {
		return fmt.Errorf("failed to create rocketmq producer: %w", err)
	}

	if err = p.producer.Start(); err != nil {
		return fmt.Errorf("failed to start rocketmq producer: %w", err)
	}

	fmt.Printf("[event-producer] started, nameServer=%s, group=%s\n", p.Config.NameServer, p.Config.GroupName)
	return nil
}

// Publish 发布消息到RocketMQ
// topic: 消息主题
// tag: 消息标签（用于消费端过滤）
// payload: 任意struct，JSON序列化后作为消息体
func (p *RMQPublisher) Publish(topic string, tag string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg := &primitive.Message{
		Topic: topic,
		Body:  body,
	}
	msg.WithTag(tag)

	result, err := p.producer.SendSync(context.Background(), msg)
	if err != nil {
		return fmt.Errorf("failed to publish [topic=%s, tag=%s]: %w", topic, tag, err)
	}

	fmt.Printf("[event-producer] published: topic=%s, tag=%s, msgId=%s\n", topic, tag, result.MsgID)
	return nil
}

// Close 实现 inji.Closeable 接口
func (p *RMQPublisher) Close() {
	if p.producer != nil {
		p.producer.Shutdown()
	}
	fmt.Println("[event-producer] shutdown")
}
