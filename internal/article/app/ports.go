package app

import "context"

// TransactionManager 定义 Article 将来扩展本地事务用例时使用的最小事务能力。
type TransactionManager interface {
	// WithinTransaction 在同一个本地事务中执行回调。
	WithinTransaction(ctx context.Context, callback func(context.Context) error) error
}
