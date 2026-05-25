package consumer

// RMQConfig RabbitMQ消费者连接配置
// 当前先写死，后期从配置中心获取
type RMQConfig struct {
	URL          string // amqp://user:pass@host:port/vhost
	Exchange     string // 交换机名称
	ExchangeType string // 交换机类型 (direct/topic/fanout)
	RoutingKey   string // 路由键（支持topic通配符）
	Queue        string // 队列名称
	PrefetchCnt  int    // 预取数量
}
