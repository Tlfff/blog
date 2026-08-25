package transaction

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

var (
	// ErrNilDatabase 表示事务协调器没有可用的数据库连接。
	ErrNilDatabase = errors.New("事务协调器未配置数据库连接")
	// ErrNilCallback 表示事务协调器没有收到执行函数。
	ErrNilCallback = errors.New("事务协调器未配置执行函数")
)

// Manager 负责在本地 MySQL 连接上协调 Application 用例事务。
type Manager struct {
	db *gorm.DB // 根 GORM 数据库连接
}

// NewManager 创建本地事务协调器。
func NewManager(db *gorm.DB) (*Manager, error) {
	if db == nil {
		return nil, ErrNilDatabase
	}
	return &Manager{db: db}, nil
}

// WithinTransaction 在本地事务中执行回调，并向下游传递事务上下文。
func (m *Manager) WithinTransaction(ctx context.Context, callback func(context.Context) error) error {
	if m == nil || m.db == nil {
		return ErrNilDatabase
	}
	if callback == nil {
		return ErrNilCallback
	}

	// 1. 如果调用方已经处于当前本地事务，只复用事务连接，避免嵌套事务改变语义
	if _, ok := DBFromContext(ctx); ok {
		return callback(ctx)
	}

	// 2. 开启 GORM 事务，并把事务连接放入派生上下文
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return callback(WithDB(ctx, tx))
	})
	if err != nil {
		return fmt.Errorf("执行本地事务失败: %w", err)
	}
	return nil
}
