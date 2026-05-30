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
	configclient "ttuser/config-client/client"
	"ttuser/pkg/trace"
)

// RMQConfig RocketMQ消费者配置
// Start时从配置中心获取
type RMQConfig struct {
	ServiceName string `inject:"serverName"`
	NameServer  string
}

// Start 实现 inji.Startable 接口
func (c *RMQConfig) Start() error {
	var rmqConf struct {
		NameServer string `json:"name_server"`
	}
	svc := c.ServiceName
	if svc == "" {
		return fmt.Errorf("[rmqConfig] ServiceName is empty, verify inject tag")
	}
	if err := configclient.LoadFile(svc, "rocketmq.json", &rmqConf); err != nil {
		return fmt.Errorf("[rmqConfig] load rocketmq config failed: %w", err)
	}
	c.NameServer = rmqConf.NameServer
	fmt.Printf("[rmqConfig] initialized: nameServer=%s\n", c.NameServer)
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

	fmt.Printf("[async-handler] started, topicGroups=%d\n", len(engine.Groups))
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

	// 订阅topic
	tagExpr := group.TagExpression()
	selector := consumer.MessageSelector{
		Type:       consumer.TAG,
		Expression: tagExpr,
	}

	currentGroup := group
	err = c.Subscribe(group.Topic, selector, func(_ context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
		for _, msg := range msgs {
			tag := msg.GetTags()
			handler, ok := currentGroup.GetHandler(tag)
			if !ok {
				log.Printf("[async-handler] no handler for tag=%s, topic=%s, skip", tag, msg.Topic)
				continue
			}

			// 从message key提取trace_id，构建带trace_id的context
			traceID := msg.GetKeys()
			if traceID == "" {
				traceID = trace.NewTraceID()
			}
			ctx := trace.WithTraceID(context.Background(), traceID)

			if err := handler.Handle(ctx, msg.Body); err != nil {
				log.Printf("[async-handler] handle error: topic=%s, tag=%s, trace_id=%s, msgId=%s, err=%v",
					msg.Topic, tag, traceID, msg.MsgId, err)
				return consumer.ConsumeRetryLater, nil
			}
			log.Printf("[async-handler] consumed: topic=%s, tag=%s, trace_id=%s, msgId=%s",
				msg.Topic, tag, traceID, msg.MsgId)
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
	log.Printf("[async-handler] subscribed: group=%s, topic=%s, tags=%s", group.ConsumerGroup, group.Topic, tagExpr)
	return nil
}

// Close 实现 inji.Closeable 接口
func (s *ConsumerServer) Close() {
	for _, c := range s.consumers {
		if err := c.Shutdown(); err != nil {
			log.Printf("[async-handler] consumer shutdown error: %v", err)
		}
	}
	fmt.Println("[async-handler] shutdown")
}
