package sp

import (
	"sync"

	"github.com/teou/inji"

	"ttuser/auth-server/internal/dao"
	"ttuser/auth-server/internal/service"
	"ttuser/auth-server/pkg/token"
	"ttuser/auth-server/server"
	"ttuser/data-store/engine"
)

// ServiceProvider 聚合所有服务依赖
// 字段顺序即创建顺序，被依赖的放前面
// ProcMysql 使用接口类型，需要通过 implmap 注册具体实现
type ServiceProvider struct {
	ProcMysql   engine.IMysqlClient      `inject:"procMysqlClient"`
	UserDAO     *dao.UserDAO             `inject:"userDAO"`
	TokenDAO    *dao.TokenDAO            `inject:"tokenDAO"`
	TokenMgr    *token.JWTManager        `inject:"tokenManager"`
	AuthService *service.AuthServiceImpl `inject:"authService"`
	GRPCServer  *server.AuthGRPCServer   `inject:"grpcServer"`
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
		instance = obj.(*ServiceProvider)
	})
}

func Get() *ServiceProvider {
	return instance
}
