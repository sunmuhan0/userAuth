package sp

import (
	"sync"

	"github.com/teou/inji"

	"ttuser/config-server/server"
)

// ServiceProvider 配置中心服务依赖聚合
type ServiceProvider struct {
	HTTPServer *server.HTTPServer `inject:"httpServer"`
}

var (
	instance *ServiceProvider
	once     sync.Once
)

func Init() {
	obj, ok := inji.Find("serviceProvider")
	if !ok {
		panic("[config-server] serviceProvider not found in inji container")
	}
	once.Do(func() {
		instance = obj.(*ServiceProvider)
	})
}

func Get() *ServiceProvider {
	return instance
}
