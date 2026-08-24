package app

import "context"

// TransactionManager 定义 Comment 用例需要的本地事务边界。
type TransactionManager interface {
	// WithinTransaction 在同一个本地事务中执行回调。
	WithinTransaction(ctx context.Context, callback func(context.Context) error) error
}
