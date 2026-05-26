package router

// Handler 消息处理接口
type Handler interface {
	Handle(body []byte) error
}

// TopicGroup 一个topic的订阅组，管理tag → handler映射
type TopicGroup struct {
	ConsumerGroup string
	Topic         string
	handlers      map[string]Handler // tag → handler
}

// Handle 注册一个 tag 对应的 handler
func (g *TopicGroup) Handle(tag string, handler Handler) {
	g.handlers[tag] = handler
}

// GetHandler 根据tag获取handler
func (g *TopicGroup) GetHandler(tag string) (Handler, bool) {
	h, ok := g.handlers[tag]
	return h, ok
}

// GetTags 获取所有注册的tag（用于subscribe的tag expression）
func (g *TopicGroup) GetTags() []string {
	tags := make([]string, 0, len(g.handlers))
	for tag := range g.handlers {
		tags = append(tags, tag)
	}
	return tags
}

// TagExpression 生成RocketMQ的tag订阅表达式（"tagA || tagB || tagC"）
func (g *TopicGroup) TagExpression() string {
	tags := g.GetTags()
	if len(tags) == 0 {
		return "*"
	}
	expr := tags[0]
	for i := 1; i < len(tags); i++ {
		expr += " || " + tags[i]
	}
	return expr
}
