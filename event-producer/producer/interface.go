package producer

// IRmqPublisher 事件发布接口
// 任何服务需要发MQ消息，注入该接口即可
// topic: 消息主题，tag: 消息标签，key: 业务唯一标识（用于查询/去重），payload: 任意struct（JSON序列化）
type IRmqPublisher interface {
	Publish(topic string, tag string, key string, payload interface{}) error
}
