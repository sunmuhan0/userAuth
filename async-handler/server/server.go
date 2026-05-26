package server

import (
	"context"
	"fmt"
	"log"
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"

	"ttuser/async-handler/biz/register"
	"ttuser/async-handler/pkg/router"
)

// RMQConfig RocketMQ消费者配置
// 实现 inji.Startable，Start中填充配置值
// 当前写死，后期从配置中心获取
type RMQConfig struct {
	NameServer string
}

// Start 实现 inji.Startable 接口
func (c *RMQConfig) Start() error {
	c.NameServer = "127.0.0.1:9876"
	fmt.Println("[rmqConfig] initialized")
	return nil
}

// ConsumerServer RocketMQ消费服务
// 基于 router.Engine 进行 topic/tag 路由
type ConsumerServer struct {
	Config    *RMQConfig `inject:"rmqConfig"`
	consumers []rocketmq.PushConsumer
}

// Start 实现 inji.Startable 接口
func (s *ConsumerServer) Start() error {
	// 初始化路由引擎
	engine := router.NewEngine()
	if err := register.InitRouter(engine); err != nil {
		return fmt.Errorf("init router failed: %w", err)
	}

	// 为每个TopicGroup创建一个PushConsumer
	for _, group := range engine.Groups {
		if err := s.startGroup(group); err != nil {
			return err
		}
	}

	fmt.Printf("[sms-consumer] started, topicGroups=%d\n", len(engine.Groups))
	return nil
}

// startGroup 为一个TopicGroup启动RocketMQ PushConsumer
func (s *ConsumerServer) startGroup(group *router.TopicGroup) error {
	c, err := rocketmq.NewPushConsumer(
		consumer.WithNameServer([]string{s.Config.NameServer}),
		consumer.WithGroupName(group.ConsumerGroup),
		consumer.WithConsumerModel(consumer.Clustering),
		consumer.WithConsumeFromWhere(consumer.ConsumeFromLastOffset),
	)
	if err != nil {
		return fmt.Errorf("failed to create consumer [group=%s]: %w", group.ConsumerGroup, err)
	}

	// 订阅topic，tag expression由router group自动生成
	tagExpr := group.TagExpression()
	selector := consumer.MessageSelector{
		Type:       consumer.TAG,
		Expression: tagExpr,
	}

	// 闭包捕获当前group
	currentGroup := group
	err = c.Subscribe(group.Topic, selector, func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
		for _, msg := range msgs {
			tag := msg.GetTags()
			handler, ok := currentGroup.GetHandler(tag)
			if !ok {
				log.Printf("[sms-consumer] no handler for tag=%s, topic=%s, skip", tag, msg.Topic)
				continue
			}
			if err := handler.Handle(msg.Body); err != nil {
				log.Printf("[sms-consumer] handle error: topic=%s, tag=%s, keys=%s, msgId=%s, err=%v",
					msg.Topic, tag, msg.GetKeys(), msg.MsgId, err)
				return consumer.ConsumeRetryLater, nil
			}
			log.Printf("[sms-consumer] consumed: topic=%s, tag=%s, keys=%s, msgId=%s",
				msg.Topic, tag, msg.GetKeys(), msg.MsgId)
		}
		return consumer.ConsumeSuccess, nil
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe [topic=%s, tags=%s]: %w", group.Topic, tagExpr, err)
	}

	if err = c.Start(); err != nil {
		return fmt.Errorf("failed to start consumer [group=%s]: %w", group.ConsumerGroup, err)
	}

	s.consumers = append(s.consumers, c)
	log.Printf("[sms-consumer] subscribed: group=%s, topic=%s, tags=%s", group.ConsumerGroup, group.Topic, tagExpr)
	return nil
}

// Close 实现 inji.Closeable 接口
func (s *ConsumerServer) Close() {
	for _, c := range s.consumers {
		c.Shutdown()
	}
	fmt.Println("[sms-consumer] shutdown")
}
