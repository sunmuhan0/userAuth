package sp

import (
	"sync"

	"github.com/teou/inji"

	"ttuser/sms-consumer/internal/handler"
	"ttuser/sms-consumer/internal/sms"
	"ttuser/sms-consumer/server"
)

// ServiceProvider 聚合所有服务依赖
// 字段顺序即创建顺序，被依赖的放前面
type ServiceProvider struct {
	SMSConfig  *sms.Config              `inject:"smsConfig"`
	SMSSender  *sms.Sender              `inject:"smsSender"`
	SMSHandler *handler.SMSHandler      `inject:"smsHandler"`
	RMQConfig  *server.RMQConfig        `inject:"rmqConfig"`
	Server     *server.SMSConsumerServer `inject:"smsConsumerServer"`
}

// Start 实现 inji.Startable 接口
// 在所有字段注入完成后，注册handler到Server
func (p *ServiceProvider) Start() error {
	// 注册topic对应的handler
	p.Server.RegisterHandler("userRegisteredHandler", p.SMSHandler)
	// 以后新增handler只需在这里加一行：
	// p.Server.RegisterHandler("orderCreatedHandler", p.OrderHandler)
	return nil
}

var (
	instance *ServiceProvider
	once     sync.Once
)

func Init() {
	obj, ok := inji.Find("serviceProvider")
	if !ok {
		panic("[sms-consumer] serviceProvider not found in inji container")
	}
	once.Do(func() {
		instance = obj.(*ServiceProvider)
	})
}

func Get() *ServiceProvider {
	return instance
}
