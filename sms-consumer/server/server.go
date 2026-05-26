package server

import (
	"context"
	"fmt"
	"log"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
)

// IMessageHandler 消息处理接口
// 不同topic的handler实现该接口
type IMessageHandler interface {
	Handle(body []byte) error
}

// Subscription 订阅配置：topic + tag + handler名称
type Subscription struct {
	Topic       string
	Tag         string // 支持 "tagA || tagB" 或 "*" 表示全部
	HandlerName string // 对应注入的handler名称
}

// RMQConfig SMS消费者RocketMQ配置
// 支持多topic订阅，每个topic指定handler
// 实现 inji.Startable，Start中填充配置值
// 当前写死，后期从配置中心获取
type RMQConfig struct {
	NameServer    string
	ConsumerGroup string
	Subscriptions []Subscription
}

// Start 实现 inji.Startable 接口
func (c *RMQConfig) Start() error {
	c.NameServer = "127.0.0.1:9876"
	c.ConsumerGroup = "sms-consumer-group"
	c.Subscriptions = []Subscription{
		{Topic: "UserTopic", Tag: "registered", HandlerName: "userRegisteredHandler"},
		// 以后新增订阅只需在这里加一行：
		// {Topic: "OrderTopic", Tag: "created", HandlerName: "orderCreatedHandler"},
	}
	fmt.Println("[rmqConsumerConfig] initialized")
	return nil
}

// SMSConsumerServer 短信消费服务
// 基于RocketMQ PushConsumer，支持多topic订阅，按topic路由到对应handler
type SMSConsumerServer struct {
	Config   *RMQConfig         `inject:"rmqConfig"`
	Handlers map[string]IMessageHandler
	consumer rocketmq.PushConsumer
}

// RegisterHandler 注册topic对应的handler
func (s *SMSConsumerServer) RegisterHandler(name string, h IMessageHandler) {
	if s.Handlers == nil {
		s.Handlers = make(map[string]IMessageHandler)
	}
	s.Handlers[name] = h
}

// Start 实现 inji.Startable 接口
func (s *SMSConsumerServer) Start() error {
	var err error
	s.consumer, err = rocketmq.NewPushConsumer(
		consumer.WithNameServer([]string{s.Config.NameServer}),
		consumer.WithGroupName(s.Config.ConsumerGroup),
		consumer.WithConsumerModel(consumer.Clustering),
		consumer.WithConsumeFromWhere(consumer.ConsumeFromLastOffset),
	)
	if err != nil {
		return fmt.Errorf("failed to create rocketmq consumer: %w", err)
	}

	// 注册所有订阅，每个subscription绑定对应的handler
	for _, sub := range s.Config.Subscriptions {
		h, ok := s.Handlers[sub.HandlerName]
		if !ok {
			return fmt.Errorf("handler [%s] not registered for topic [%s]", sub.HandlerName, sub.Topic)
		}

		selector := consumer.MessageSelector{
			Type:       consumer.TAG,
			Expression: sub.Tag,
		}
		// 闭包捕获当前handler
		currentHandler := h
		currentSub := sub
		err = s.consumer.Subscribe(currentSub.Topic, selector, func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
			for _, msg := range msgs {
				if err := currentHandler.Handle(msg.Body); err != nil {
					log.Printf("[sms-consumer] handle error: topic=%s, tag=%s, msgId=%s, err=%v",
						msg.Topic, msg.GetTags(), msg.MsgId, err)
					return consumer.ConsumeRetryLater, nil
				}
				log.Printf("[sms-consumer] consumed: topic=%s, tag=%s, msgId=%s", msg.Topic, msg.GetTags(), msg.MsgId)
			}
			return consumer.ConsumeSuccess, nil
		})
		if err != nil {
			return fmt.Errorf("failed to subscribe [topic=%s, tag=%s]: %w", sub.Topic, sub.Tag, err)
		}
		log.Printf("[sms-consumer] subscribed: topic=%s, tag=%s, handler=%s", sub.Topic, sub.Tag, sub.HandlerName)
	}

	if err = s.consumer.Start(); err != nil {
		return fmt.Errorf("failed to start rocketmq consumer: %w", err)
	}

	fmt.Printf("[sms-consumer] started, group=%s, subscriptions=%d\n", s.Config.ConsumerGroup, len(s.Config.Subscriptions))
	return nil
}

// Close 实现 inji.Closeable 接口
func (s *SMSConsumerServer) Close() {
	if s.consumer != nil {
		s.consumer.Shutdown()
	}
	fmt.Println("[sms-consumer-server] shutdown")
}
