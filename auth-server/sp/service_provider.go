package sp

import (
	"sync"

	"github.com/teou/inji"

	"ttuser/auth-server/server"
)

// ServiceProvider 聚合所有服务依赖
// 只声明外部需要访问的顶层组件
// 中间依赖（DAO、TokenMgr、EventPublisher等）由inji自动递归创建
type ServiceProvider struct {
	GRPCServer *server.AuthGRPCServer `inject:"grpcServer"`
}

var (
	instance *ServiceProvider
	once     sync.Once
)

func Init() {
	obj, ok := inji.Find("serviceProvider")
	if !ok {
		panic("[auth-server] serviceProvider not found in inji container")
	}
	once.Do(func() {
		var ok bool
		instance, ok = obj.(*ServiceProvider)
		if !ok {
			panic("[auth-server] serviceProvider is not *ServiceProvider type")
		}
	})
}

func Get() *ServiceProvider {
	if instance == nil {
		panic("[auth-server] ServiceProvider not initialized: call Init() first")
	}
	return instance
}
