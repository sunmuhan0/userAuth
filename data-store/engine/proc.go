package engine

import (
	"fmt"
	"reflect"

	"github.com/teou/implmap"
	"github.com/teou/inji"

	configclient "ttuser/config-client/client"
)

func init() {
	implmap.Add("procMysqlClient", reflect.TypeOf((*ProcMysqlClient)(nil)))
}

type ProcMysqlClient struct {
	BaseMysqlClient
}

func (c *ProcMysqlClient) Start() error {
	svc := "auth-server"
	if v, ok := inji.Find("serverName"); ok {
		svc = v.(string)
	}
	var mysqlConf struct {
		DSN string `json:"dsn"`
	}
	if err := configclient.LoadFile(svc, "mysql.json", &mysqlConf); err != nil {
		return fmt.Errorf("[ProcMysqlClient] load mysql config failed: %w", err)
	}
	if err := c.Connect(mysqlConf.DSN); err != nil {
		return fmt.Errorf("[ProcMysqlClient] connect failed: %w", err)
	}
	fmt.Println("[ProcMysqlClient] connected to mysql")
	return nil
}

// Close 实现 inji.Closeable 接口
func (c *ProcMysqlClient) Close() {
	c.CloseDB()
	fmt.Println("[ProcMysqlClient] connection closed")
}
