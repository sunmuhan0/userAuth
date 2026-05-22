package engine

import (
	"context"
	"database/sql"
)

// TransactionExec 事务执行函数
type TransactionExec func(tx *sql.Tx) (int64, error)

//go:generate mockgen -source $GOFILE -destination "mock_interface.go" -package "${GOPACKAGE}"

// IMysqlClient MySQL客户端接口
// 各业务MySQL实例都实现此接口
type IMysqlClient interface {
	Start() error
	Close()
	QueryList(dataType interface{}, query string, args ...interface{}) ([]interface{}, error)
	Query(dataType interface{}, query string, args ...interface{}) (interface{}, error)
	GetCount(query string, args ...interface{}) (int64, error)
	Update(tableName string, d interface{}, primaryKeys map[string]interface{}, fieldsToUpdate []string) error
	Delete(tableName string, condition map[string]interface{}) (int64, error)
	Execute(sql string, args ...interface{}) (int64, error)
	ExecContext(ctx context.Context, sql string, args ...interface{}) (int64, error)
	ExecTransaction(transactionExec TransactionExec) (int64, error)
	InsertOrUpdateOnDup(tableName string, d interface{}, updateFields []string, ignoreFields ...string) (int64, error)
	Add(tableName string, d interface{}, ondupUpdate bool) error
	AddAndRetLastId(tableName string, d interface{}, ignoreFields ...string) (int64, error)
}
