// Package transaction 提供本地数据库事务协调能力。
package transaction

import (
	"context"

	"gorm.io/gorm"
)

type contextKey struct{}

// WithDB 将当前 GORM 事务绑定到派生上下文。
func WithDB(ctx context.Context, db *gorm.DB) context.Context {
	return context.WithValue(ctx, contextKey{}, db)
}

// DBFromContext 从上下文中读取当前 GORM 事务。
func DBFromContext(ctx context.Context) (*gorm.DB, bool) {
	db, ok := ctx.Value(contextKey{}).(*gorm.DB)
	return db, ok && db != nil
}

// DB 返回上下文中的事务连接；不存在事务时返回 fallback。
func DB(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	if db, ok := DBFromContext(ctx); ok {
		return db
	}
	return fallback
}
