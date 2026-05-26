package server

import (
	"context"
	"fmt"
	"log"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"

	"ttuser/sms-consumer/internal/handler"
)

// Subscription 订阅配置：topic + tag
type Subscription struct {
	Topic string
	Tag   string // 支持 "tagA || tagB" 或 "*" 表示全部
}

// RMQConfig SMS消费者RocketMQ配置
// 支持多topic订阅
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
		{Topic: "UserTopic", Tag: "registered"},
		// 以后新增订阅只需在这里加一行：
		// {Topic: "OrderTopic", Tag: "created"},
	}
	fmt.Println("[rmqConsumerConfig] initialized")
	return nil
}

// SMSConsumerServer 短信消费服务
// 基于RocketMQ PushConsumer，支持多topic订阅
type SMSConsumerServer struct {
	Config   *RMQConfig          `inject:"rmqConfig"`
	Handler  *handler.SMSHandler `inject:"smsHandler"`
	consumer rocketmq.PushConsumer
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

	// 注册所有订阅
	for _, sub := range s.Config.Subscriptions {
		selector := consumer.MessageSelector{
			Type:       consumer.TAG,
			Expression: sub.Tag,
		}
		topicCopy := sub.Topic
		err = s.consumer.Subscribe(topicCopy, selector, func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
			for _, msg := range msgs {
				if err := s.Handler.Handle(msg.Body); err != nil {
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
		log.Printf("[sms-consumer] subscribed: topic=%s, tag=%s", sub.Topic, sub.Tag)
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
