package producer

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	rmqproducer "github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/teou/implmap"

	"ttuser/pkg/trace"
)

func init() {
	// 注册 IRmqPublisher 接口的实现
	implmap.Add("eventPublisher", reflect.TypeOf((*EventRMQPublisher)(nil)))
}

// EventRMQPublisher 基于RocketMQ的事件发布实现
// 通过inji注入Config，Start()时自动创建producer
// 业务方注入 *EventRMQPublisher 即可直接使用
type EventRMQPublisher struct {
	Config   *RMQConfig `inject:"rmqProducerConfig"`
	producer rocketmq.Producer
}

// Start 实现 inji.Startable 接口，创建并启动RocketMQ producer
func (p *EventRMQPublisher) Start() error {
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
// 自动从ctx提取trace_id作为message key（用于RocketMQ控制台按trace_id检索消息）
// topic: 消息主题
// tag: 消息标签（用于消费端过滤）
// payload: 任意struct，JSON序列化后作为消息体
func (p *EventRMQPublisher) Publish(ctx context.Context, topic string, tag string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	traceID := trace.GetTraceID(ctx)

	msg := &primitive.Message{
		Topic: topic,
		Body:  body,
	}
	msg.WithTag(tag)
	if traceID != "" {
		msg.WithKeys([]string{traceID})
	}

	result, err := p.producer.SendSync(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to publish [topic=%s, tag=%s, trace_id=%s]: %w", topic, tag, traceID, err)
	}

	fmt.Printf("[event-producer] published: topic=%s, tag=%s, trace_id=%s, msgId=%s\n", topic, tag, traceID, result.MsgID)
	return nil
}

// Close 实现 inji.Closeable 接口
func (p *EventRMQPublisher) Close() {
	if p.producer != nil {
		p.producer.Shutdown()
	}
	fmt.Println("[event-producer] shutdown")
}
