package register

import (
	"ttuser/sms-consumer/biz/actions"
	"ttuser/sms-consumer/pkg/router"
)

// 消费者组和Topic常量
const (
	SMSConsumerGroup = "sms-consumer-group"
	TopicUser        = "UserTopic"
)

// InitRouter 初始化消息路由
// 统一注册所有 topic/tag/handler 映射
func InitRouter(engine *router.Engine) error {
	// === UserTopic 订阅组 ===
	userGroup := engine.NewTopicGroup(SMSConsumerGroup, TopicUser)

	// tag "registered" → 用户注册，发短信
	h, err := router.WrapHandleFunc(actions.UserRegistered)
	if err != nil {
		return err
	}
	userGroup.Handle("registered", h)

	// 以后新增：
	// h2, _ := router.WrapHandleFunc(actions.UserUpdated)
	// userGroup.Handle("updated", h2)

	// === 新Topic订阅组示例 ===
	// orderGroup := engine.NewTopicGroup("sms-order-group", "OrderTopic")
	// h3, _ := router.WrapHandleFunc(actions.OrderCreated)
	// orderGroup.Handle("created", h3)

	return nil
}
