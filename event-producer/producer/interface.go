package producer

import "context"

// IRmqPublisher 事件发布接口
// 任何服务需要发MQ消息，注入该接口即可
// ctx: 上下文（自动提取trace_id作为message key）
// topic: 消息主题，tag: 消息标签，payload: 任意struct（JSON序列化）
type IRmqPublisher interface {
	Publish(ctx context.Context, topic string, tag string, payload interface{}) error
}
