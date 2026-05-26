package sp

import (
	"sync"

	"github.com/teou/inji"

	"ttuser/proc/internal/manager"
)

// ServiceProvider 聚合所有服务依赖，通过inji自动注入
// 其他包通过 sp.Get() 获取单例来访问各 manager
type ServiceProvider struct {
	AuthManager *manager.AuthManager `inject:"authManager"`
}

var (
	instance *ServiceProvider
	once     sync.Once
)

// Init 从inji容器中获取ServiceProvider单例
func Init() {
	obj, ok := inji.Find("serviceProvider")
	if !ok {
		panic("[proc] serviceProvider not found in inji container")
	}
	once.Do(func() {
		var ok bool
		instance, ok = obj.(*ServiceProvider)
		if !ok {
			panic("[proc] serviceProvider is not *ServiceProvider type")
		}
	})
}

// Get 获取ServiceProvider单例
func Get() *ServiceProvider {
	if instance == nil {
		panic("[proc] ServiceProvider not initialized: call Init() first")
	}
	return instance
}
