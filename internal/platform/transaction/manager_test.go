package transaction

import (
	"context"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// transactionRecord 是事务测试使用的第一张持久化表。
type transactionRecord struct {
	ID    uint64 `gorm:"primaryKey"` // 记录唯一标识
	Value string // 记录内容
}

// transactionAudit 是事务测试使用的第二张持久化表。
type transactionAudit struct {
	ID    uint64 `gorm:"primaryKey"` // 审计记录唯一标识
	Value string // 审计内容
}

// openTestDB 创建只使用单连接的内存数据库，避免 SQLite 内存库连接隔离。
func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取测试数据库连接失败: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(&transactionRecord{}, &transactionAudit{}); err != nil {
		t.Fatalf("创建测试表失败: %v", err)
	}
	return db
}

// countRecords 查询两张测试表的记录数量。
func countRecords(t *testing.T, db *gorm.DB) (int64, int64) {
	t.Helper()

	var records int64
	if err := db.Model(&transactionRecord{}).Count(&records).Error; err != nil {
		t.Fatalf("查询事务记录数量失败: %v", err)
	}

	var audits int64
	if err := db.Model(&transactionAudit{}).Count(&audits).Error; err != nil {
		t.Fatalf("查询审计记录数量失败: %v", err)
	}
	return records, audits
}

// TestManagerCommit 验证事务回调成功时数据能够共同提交。
func TestManagerCommit(t *testing.T) {
	db := openTestDB(t)
	manager, err := NewManager(db)
	if err != nil {
		t.Fatalf("创建事务协调器失败: %v", err)
	}

	err = manager.WithinTransaction(context.Background(), func(ctx context.Context) error {
		tx := DB(ctx, db)
		if err := tx.WithContext(ctx).Create(&transactionRecord{Value: "committed"}).Error; err != nil {
			return err
		}
		return tx.WithContext(ctx).Create(&transactionAudit{Value: "committed"}).Error
	})
	if err != nil {
		t.Fatalf("事务提交失败: %v", err)
	}

	records, audits := countRecords(t, db)
	if records != 1 || audits != 1 {
		t.Fatalf("事务提交结果不正确: records=%d audits=%d", records, audits)
	}
}

// TestManagerRollback 验证事务回调返回错误时所有修改均回滚。
func TestManagerRollback(t *testing.T) {
	db := openTestDB(t)
	manager, err := NewManager(db)
	if err != nil {
		t.Fatalf("创建事务协调器失败: %v", err)
	}
	wantErr := errors.New("模拟业务失败")

	err = manager.WithinTransaction(context.Background(), func(ctx context.Context) error {
		tx := DB(ctx, db)
		if err := tx.WithContext(ctx).Create(&transactionRecord{Value: "rolled-back"}).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Create(&transactionAudit{Value: "rolled-back"}).Error; err != nil {
			return err
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("事务错误链不正确: got=%v want=%v", err, wantErr)
	}

	records, audits := countRecords(t, db)
	if records != 0 || audits != 0 {
		t.Fatalf("事务回滚结果不正确: records=%d audits=%d", records, audits)
	}
}

// TestDBFallback verifies that the transaction helper returns the fallback connection outside a transaction.
func TestDBFallback(t *testing.T) {
	db := openTestDB(t)
	if got := DB(context.Background(), db); got != db {
		t.Fatal("无事务上下文时没有返回 fallback 数据库连接")
	}

	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("开启测试事务失败: %v", tx.Error)
	}
	defer tx.Rollback()

	ctx := WithDB(context.Background(), tx)
	if got := DB(ctx, db); got != tx {
		t.Fatal("有事务上下文时没有返回当前事务连接")
	}
}

// TestManagerSharesTransactionAcrossRepositories 验证两个持久化操作共享同一个事务上下文。
func TestManagerSharesTransactionAcrossRepositories(t *testing.T) {
	db := openTestDB(t)
	manager, err := NewManager(db)
	if err != nil {
		t.Fatalf("创建事务协调器失败: %v", err)
	}

	err = manager.WithinTransaction(context.Background(), func(ctx context.Context) error {
		// 模拟两个上下文 Repository 只从事务上下文取得连接，不接收 *gorm.DB 参数。
		articleSide := DB(ctx, db)
		likeSide := DB(ctx, db)
		if articleSide != likeSide {
			return errors.New("两个 Repository 没有共享同一个事务连接")
		}
		if err := likeSide.WithContext(ctx).Create(&transactionRecord{Value: "like"}).Error; err != nil {
			return err
		}
		return articleSide.WithContext(ctx).Create(&transactionAudit{Value: "article-counter"}).Error
	})
	if err != nil {
		t.Fatalf("跨 Repository 事务执行失败: %v", err)
	}

	records, audits := countRecords(t, db)
	if records != 1 || audits != 1 {
		t.Fatalf("跨 Repository 事务提交结果不正确: records=%d audits=%d", records, audits)
	}
}
