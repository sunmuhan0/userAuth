package engine

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// BaseMysqlClient 通用MySQL客户端实现
// 各具体MySQL服务（如ProcMysqlClient）内嵌此struct复用实现
type BaseMysqlClient struct {
	db *sql.DB
}

// Connect 建立连接
func (c *BaseMysqlClient) Connect(dsn string) error {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open mysql: %w", err)
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping mysql: %w", err)
	}

	c.db = db
	return nil
}

// CloseDB 关闭连接
func (c *BaseMysqlClient) CloseDB() {
	if c.db != nil {
		c.db.Close()
	}
}

// QueryList 查询列表
func (c *BaseMysqlClient) QueryList(dataType interface{}, query string, args ...interface{}) ([]interface{}, error) {
	rows, err := c.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return c.scanRows(rows, dataType)
}

// Query 查询单条
func (c *BaseMysqlClient) Query(dataType interface{}, query string, args ...interface{}) (interface{}, error) {
	list, err := c.QueryList(dataType, query, args...)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return list[0], nil
}

// GetCount 获取记录数
func (c *BaseMysqlClient) GetCount(query string, args ...interface{}) (int64, error) {
	var count int64
	err := c.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// Update 更新记录
func (c *BaseMysqlClient) Update(tableName string, d interface{}, primaryKeys map[string]interface{}, fieldsToUpdate []string) error {
	setClauses := make([]string, 0, len(fieldsToUpdate))
	args := make([]interface{}, 0)

	v := reflect.ValueOf(d)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()

	fieldMap := make(map[string]interface{})
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		dbTag := field.Tag.Get("db")
		if dbTag == "" {
			dbTag = strings.ToLower(field.Name)
		}
		fieldMap[dbTag] = v.Field(i).Interface()
	}

	for _, f := range fieldsToUpdate {
		if val, ok := fieldMap[f]; ok {
			setClauses = append(setClauses, fmt.Sprintf("`%s` = ?", f))
			args = append(args, val)
		}
	}

	if len(setClauses) == 0 {
		return nil
	}

	whereClauses := make([]string, 0, len(primaryKeys))
	for k, v := range primaryKeys {
		whereClauses = append(whereClauses, fmt.Sprintf("`%s` = ?", k))
		args = append(args, v)
	}

	query := fmt.Sprintf("UPDATE `%s` SET %s WHERE %s",
		tableName, strings.Join(setClauses, ", "), strings.Join(whereClauses, " AND "))

	_, err := c.db.Exec(query, args...)
	return err
}

// Delete 删除记录
func (c *BaseMysqlClient) Delete(tableName string, condition map[string]interface{}) (int64, error) {
	whereClauses := make([]string, 0, len(condition))
	args := make([]interface{}, 0, len(condition))
	for k, v := range condition {
		whereClauses = append(whereClauses, fmt.Sprintf("`%s` = ?", k))
		args = append(args, v)
	}

	query := fmt.Sprintf("DELETE FROM `%s` WHERE %s", tableName, strings.Join(whereClauses, " AND "))
	result, err := c.db.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Execute 执行SQL
func (c *BaseMysqlClient) Execute(sqlStr string, args ...interface{}) (int64, error) {
	result, err := c.db.Exec(sqlStr, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// ExecContext 带context执行SQL
func (c *BaseMysqlClient) ExecContext(ctx context.Context, sqlStr string, args ...interface{}) (int64, error) {
	result, err := c.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// ExecTransaction 事务执行
func (c *BaseMysqlClient) ExecTransaction(transactionExec TransactionExec) (int64, error) {
	tx, err := c.db.Begin()
	if err != nil {
		return 0, err
	}
	rowsAffected, err := transactionExec(tx)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	return rowsAffected, tx.Commit()
}

// InsertOrUpdateOnDup 插入，遇到唯一键冲突时更新指定字段
// updateFields: 冲突时更新的字段列表（为空则不加 ON DUPLICATE KEY UPDATE）
// ignoreFields: 插入时跳过的字段（如自增主键）
func (c *BaseMysqlClient) InsertOrUpdateOnDup(tableName string, d interface{}, updateFields []string, ignoreFields ...string) (int64, error) {
	columns, args := c.structToColumnsValues(d, ignoreFields)

	placeholders := make([]string, len(columns))
	for i := range columns {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)",
		tableName, strings.Join(columns, ", "), strings.Join(placeholders, ", "))

	if len(updateFields) > 0 {
		updateClauses := make([]string, 0, len(updateFields))
		for _, f := range updateFields {
			updateClauses = append(updateClauses, fmt.Sprintf("`%s` = VALUES(`%s`)", f, f))
		}
		query += " ON DUPLICATE KEY UPDATE " + strings.Join(updateClauses, ", ")
	}

	result, err := c.db.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Add 简单插入
func (c *BaseMysqlClient) Add(tableName string, d interface{}, ondupUpdate bool) error {
	columns, args := c.structToColumnsValues(d, nil)

	placeholders := make([]string, len(columns))
	for i := range columns {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)",
		tableName, strings.Join(columns, ", "), strings.Join(placeholders, ", "))

	if ondupUpdate {
		updateClauses := make([]string, 0, len(columns))
		for _, col := range columns {
			updateClauses = append(updateClauses, fmt.Sprintf("%s = VALUES(%s)", col, col))
		}
		query += " ON DUPLICATE KEY UPDATE " + strings.Join(updateClauses, ", ")
	}

	_, err := c.db.Exec(query, args...)
	return err
}

// AddAndRetLastId 插入并返回自增主键ID
// ignoreFields: 插入时跳过的字段（如自增主键本身）
func (c *BaseMysqlClient) AddAndRetLastId(tableName string, d interface{}, ignoreFields ...string) (int64, error) {
	columns, args := c.structToColumnsValues(d, ignoreFields)

	placeholders := make([]string, len(columns))
	for i := range columns {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)",
		tableName, strings.Join(columns, ", "), strings.Join(placeholders, ", "))

	result, err := c.db.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// structToColumnsValues struct转列名和值
func (c *BaseMysqlClient) structToColumnsValues(d interface{}, ignoreFields []string) ([]string, []interface{}) {
	v := reflect.ValueOf(d)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()

	ignoreMap := make(map[string]bool)
	for _, f := range ignoreFields {
		ignoreMap[f] = true
	}

	columns := make([]string, 0)
	values := make([]interface{}, 0)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		dbTag := field.Tag.Get("db")
		if dbTag == "" || dbTag == "-" {
			continue
		}
		if ignoreMap[dbTag] {
			continue
		}
		columns = append(columns, fmt.Sprintf("`%s`", dbTag))
		values = append(values, v.Field(i).Interface())
	}

	return columns, values
}

// scanRows 将结果映射到struct列表
func (c *BaseMysqlClient) scanRows(rows *sql.Rows, dataType interface{}) ([]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	t := reflect.TypeOf(dataType)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	fieldMap := make(map[string]int)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		dbTag := field.Tag.Get("db")
		if dbTag == "" {
			dbTag = strings.ToLower(field.Name)
		}
		fieldMap[dbTag] = i
	}

	result := make([]interface{}, 0)

	for rows.Next() {
		elem := reflect.New(t).Elem()
		scanDest := make([]interface{}, len(columns))

		for i, col := range columns {
			if fieldIdx, ok := fieldMap[col]; ok {
				scanDest[i] = elem.Field(fieldIdx).Addr().Interface()
			} else {
				var ignore interface{}
				scanDest[i] = &ignore
			}
		}

		if err := rows.Scan(scanDest...); err != nil {
			return nil, err
		}

		result = append(result, elem.Addr().Interface())
	}

	return result, rows.Err()
}
