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
func Init() error {
	obj, ok := inji.Find("serviceProvider")
	if !ok {
		return nil
	}
	once.Do(func() {
		instance = obj.(*ServiceProvider)
	})
	return nil
}

// Get 获取ServiceProvider单例
func Get() *ServiceProvider {
	return instance
}
