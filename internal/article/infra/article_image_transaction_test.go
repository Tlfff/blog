package infra

import (
	domaincontent "blog/internal/article/domain"
	"blog/internal/article/infra/model"
	platformtransaction "blog/internal/platform/transaction"
	"context"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestArticleAndImagesShareTransaction 验证文章创建和图片绑定共享同一事务。
func TestArticleAndImagesShareTransaction(t *testing.T) {
	// 1. 创建包含文章表和图片表的内存数据库
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.Article{}, &model.ArticleImage{}); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}
	articles := NewArticleRepository(db)
	images := NewArticleImageRepository(db)
	manager, err := platformtransaction.NewManager(db)
	if err != nil {
		t.Fatalf("创建事务协调器失败: %v", err)
	}

	// 2. 先创建未绑定图片，再在事务内创建文章并绑定后主动回滚
	ctx := context.Background()
	image := &domaincontent.ArticleImage{ObjectKey: "article/img/2026/08/transaction.png"}
	if err := images.Create(ctx, image); err != nil {
		t.Fatalf("创建未绑定图片失败: %v", err)
	}
	rollbackErr := errors.New("rollback article image transaction")
	err = manager.WithinTransaction(ctx, func(txCtx context.Context) error {
		article, err := domaincontent.NewArticle(100, "文章", "![image](image://1)", "", domaincontent.StatusDraft.Int8())
		if err != nil {
			return err
		}
		if err := articles.Create(txCtx, article); err != nil {
			return err
		}
		rows, err := images.BindArticle(txCtx, []uint64{image.ID}, article.ID)
		if err != nil {
			return err
		}
		if rows != 1 {
			t.Fatalf("绑定图片影响行数错误: %d", rows)
		}
		return rollbackErr
	})
	if !errors.Is(err, rollbackErr) {
		t.Fatalf("事务错误不正确: %v", err)
	}

	// 3. 验证文章写入和图片绑定同时回滚
	if _, err := articles.FindByID(ctx, 1); !errors.Is(err, domaincontent.ErrArticleNotFound) {
		t.Fatalf("回滚后文章仍存在: %v", err)
	}
	storedImages, err := images.FindByIDs(ctx, []uint64{image.ID})
	if err != nil || len(storedImages) != 1 || storedImages[0].ArticleID != 0 {
		t.Fatalf("回滚后图片关系错误: images=%+v err=%v", storedImages, err)
	}
}
