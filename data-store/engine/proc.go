package engine

import (
	"fmt"
	"reflect"

	"github.com/teou/implmap"
)

func init() {
	implmap.Add("procMysqlClient", reflect.TypeOf((*ProcMysqlClient)(nil)))
}

// proc 服务 MySQL 配置，后续从配置中心获取
const procDSN = "root:123456@tcp(localhost:3306)/ttuser?charset=utf8mb4&parseTime=true&loc=Local"

// ProcMysqlClient proc 服务的 MySQL 客户端
// 内嵌 BaseMysqlClient 复用通用实现
// 通过 inji 自动注册：inject:"procMysqlClient"
type ProcMysqlClient struct {
	BaseMysqlClient
}

// Start 实现 inji.Startable 接口
func (c *ProcMysqlClient) Start() error {
	if err := c.Connect(procDSN); err != nil {
		return err
	}
	fmt.Println("[ProcMysqlClient] connected to mysql")
	return nil
}

// Close 实现 inji.Closeable 接口
func (c *ProcMysqlClient) Close() {
	c.CloseDB()
	fmt.Println("[ProcMysqlClient] connection closed")
}
