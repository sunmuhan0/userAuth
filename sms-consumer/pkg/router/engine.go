package router

// Engine 消息路由引擎，管理多个topic group
type Engine struct {
	Groups []*TopicGroup
}

// NewEngine 创建路由引擎
func NewEngine() *Engine {
	return &Engine{}
}

// NewTopicGroup 创建一个topic订阅组
func (e *Engine) NewTopicGroup(consumerGroup, topic string) *TopicGroup {
	g := &TopicGroup{
		ConsumerGroup: consumerGroup,
		Topic:         topic,
		handlers:      make(map[string]Handler),
	}
	e.Groups = append(e.Groups, g)
	return g
}
