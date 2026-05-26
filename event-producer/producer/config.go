package producer

// IRMQProducerConfig RMQ生产者配置接口
// 各业务服务实现该接口，通过inji注入到RMQPublisher
// 不同服务可以连不同的RabbitMQ、使用不同的exchange
type IRMQProducerConfig interface {
	GetURL() string
	GetExchange() string
	GetExchangeType() string
}
