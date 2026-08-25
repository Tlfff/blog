package infra

import (
	articleapp "blog/internal/article/app"
	articleinfra "blog/internal/article/infra"
	articlemodel "blog/internal/article/infra/model"
	commentapp "blog/internal/comment/app"
	commentdomain "blog/internal/comment/domain"
	commentmodel "blog/internal/comment/infra/model"
	platformtransaction "blog/internal/platform/transaction"
	"context"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// failingArticleStatistics 模拟文章评论数更新失败。
type failingArticleStatistics struct{}

// IncrementCommentCount 返回模拟错误以验证事务回滚。
func (failingArticleStatistics) IncrementCommentCount(context.Context, uint64, int64) error {
	return errors.New("模拟文章评论数更新失败")
}

// openCommentTestDB 创建评论事务测试数据库。
func openCommentTestDB(t *testing.T) *gorm.DB {
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
	if err := db.AutoMigrate(&articlemodel.Article{}, &commentmodel.Comment{}); err != nil {
		t.Fatalf("创建测试表失败: %v", err)
	}
	return db
}

// TestCreateCommentSharesArticleTransaction 验证评论和文章评论数共同提交。
func TestCreateCommentSharesArticleTransaction(t *testing.T) {
	db := openCommentTestDB(t)
	if err := db.Create(&articlemodel.Article{ID: 1, Status: 3}).Error; err != nil {
		t.Fatalf("创建测试文章失败: %v", err)
	}
	tx, _ := platformtransaction.NewManager(db)
	articleStats := articleapp.NewStatisticsService(articleinfra.NewArticleStatistics(db))
	service := commentapp.NewService(NewCommentRepository(db, articleStats), nil, tx)

	if _, err := service.CreateComment(context.Background(), 1, 0, 2, 0, "评论", "127.0.0.1"); err != nil {
		t.Fatalf("创建评论失败: %v", err)
	}
	var article articlemodel.Article
	if err := db.First(&article, 1).Error; err != nil {
		t.Fatalf("查询文章失败: %v", err)
	}
	if article.CommentCount != 1 {
		t.Fatalf("文章评论数未共同提交: %d", article.CommentCount)
	}
	var count int64
	if err := db.Model(&commentmodel.Comment{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("评论记录未提交: count=%d err=%v", count, err)
	}
}

// TestCreateCommentRollsBackWhenArticleCountFails 验证文章计数失败时评论回滚。
func TestCreateCommentRollsBackWhenArticleCountFails(t *testing.T) {
	db := openCommentTestDB(t)
	tx, _ := platformtransaction.NewManager(db)
	service := commentapp.NewService(NewCommentRepository(db, failingArticleStatistics{}), nil, tx)

	if _, err := service.CreateComment(context.Background(), 1, 0, 2, 0, "评论", "127.0.0.1"); err == nil {
		t.Fatal("文章计数失败时应返回错误")
	}
	var count int64
	if err := db.Model(&commentmodel.Comment{}).Count(&count).Error; err != nil {
		t.Fatalf("查询评论数量失败: %v", err)
	}
	if count != 0 {
		t.Fatalf("评论没有随事务回滚: %d", count)
	}
}

var _ commentdomain.ArticleStatistics = failingArticleStatistics{}
