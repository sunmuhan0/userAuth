package sp

import (
	"sync"

	"github.com/teou/inji"

	"ttuser/async-handler/internal/sms"
	"ttuser/async-handler/server"
)

// ServiceProvider 聚合所有服务依赖
// SMSSender 需要在SP中声明以确保inji在Server之前创建它（action函数通过sms.GetSender()访问）
type ServiceProvider struct {
	SMSSender *sms.Sender          `inject:"smsSender"`
	Server    *server.ConsumerServer `inject:"consumerServer"`
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
