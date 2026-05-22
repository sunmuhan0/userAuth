package sp

import (
	"sync"

	"github.com/teou/inji"

	"ttuser/auth-server/internal/service"
	"ttuser/auth-server/internal/store"
	"ttuser/auth-server/pkg/token"
	"ttuser/auth-server/server"
)

// ServiceProvider 聚合所有服务依赖
// 全部使用具体指针类型，inji 直接创建实例并调 Start() 初始化
// 字段顺序即创建顺序，被依赖的放前面
type ServiceProvider struct {
	UserStore   *store.MemoryUserStore   `inject:"userStore"`
	TokenStore  *store.MemoryTokenStore  `inject:"tokenStore"`
	TokenMgr    *token.JWTManager        `inject:"tokenManager"`
	AuthService *service.AuthServiceImpl `inject:"authService"`
	GRPCServer  *server.AuthGRPCServer   `inject:"grpcServer"`
}

var (
	instance *ServiceProvider
	once     sync.Once
)

// Init 从inji容器中获取ServiceProvider单例
func Init() {
	obj, ok := inji.Find("serviceProvider")
	if !ok {
		return
	}
	once.Do(func() {
		instance = obj.(*ServiceProvider)
	})
}

// Get 获取ServiceProvider单例
func Get() *ServiceProvider {
	return instance
}
