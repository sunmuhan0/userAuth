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
	SMSConfig  *sms.Config            `inject:"smsConfig"`
	SMSSender  *sms.Sender            `inject:"smsSender"`
	SMSHandler *handler.SMSHandler    `inject:"smsHandler"`
	RMQConfig  *server.RMQConfig      `inject:"rmqConfig"`
	Server     *server.SMSConsumerServer `inject:"smsConsumerServer"`
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
