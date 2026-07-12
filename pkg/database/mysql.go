package database

import (
	"context"
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 负责连接mysql，返回一个原生的 *gorm.DB
func NewMySQLClient(username, password, host string, port int, dbname string) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		username, password, host, port, dbname)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}

// RunTx:事务执行模板
// ctx:请求上下文，用于传递超时、链路追踪信息
// db:原始gorm.DB连接实例（来自各repository）
// txLogic:闭包函数，需要事务包裹的所有增删改逻辑
// return:事务执行中出现的任意错误，失败会自动回滚，无错误则自动提交
func RunTx(ctx context.Context, db *gorm.DB, txLogic func(*gorm.DB) error) error {
	// 1. 给数据库DB绑定当前上下文ctx
	db = db.WithContext(ctx)

	// 2. 开启GORM事务，执行传入的业务闭包txLogic
	// GORM内部机制：
	// - 若txLogic返回nil：自动COMMIT提交事务，所有修改持久化到数据库
	// - 若txLogic返回error：自动ROLLBACK回滚，本次事务所有修改全部撤销
	return db.Transaction(txLogic)
}
