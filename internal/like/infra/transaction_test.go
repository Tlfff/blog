package infra

import (
	articleapp "blog/internal/article/app"
	articleinfra "blog/internal/article/infra"
	articlemodel "blog/internal/article/infra/model"
	commentapp "blog/internal/comment/app"
	commentinfra "blog/internal/comment/infra"
	commentmodel "blog/internal/comment/infra/model"
	likeapp "blog/internal/like/app"
	likedomain "blog/internal/like/domain"
	likemodel "blog/internal/like/infra/model"
	platformtransaction "blog/internal/platform/transaction"
	"context"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// failingProjection 模拟目标点赞数更新失败。
type failingProjection struct{}

// ApplyLikeDelta 返回模拟错误。
func (failingProjection) ApplyLikeDelta(context.Context, likedomain.LikeTarget, uint64, int64) error {
	return errors.New("模拟点赞数更新失败")
}

// openLikeTestDB 创建点赞事务测试数据库。
func openLikeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&articlemodel.Article{}, &commentmodel.Comment{}, &likemodel.ArticleLike{}, &likemodel.CommentLike{}); err != nil {
		t.Fatalf("创建测试表失败: %v", err)
	}
	return db
}

// TestArticleLikeSharesCounterTransaction 验证点赞关系和文章点赞数共同提交。
func TestArticleLikeSharesCounterTransaction(t *testing.T) {
	db := openLikeTestDB(t)
	if err := db.Create(&articlemodel.Article{ID: 1, Status: 3}).Error; err != nil {
		t.Fatalf("创建测试文章失败: %v", err)
	}
	tx, _ := platformtransaction.NewManager(db)
	articleStats := articleapp.NewStatisticsService(articleinfra.NewArticleStatistics(db))
	commentStats := commentapp.NewLikeProjectionService(commentinfra.NewCommentLikeStatistics(db))
	service := likeapp.NewService(
		NewArticleLikeRepository(db), NewCommentLikeRepository(db), nil, nil,
		NewProjectionUpdater(articleStats, commentStats), tx,
	)

	if err := service.ArticleLike(context.Background(), 2, 1); err != nil {
		t.Fatalf("文章点赞失败: %v", err)
	}
	var article articlemodel.Article
	if err := db.First(&article, 1).Error; err != nil {
		t.Fatalf("查询文章失败: %v", err)
	}
	if article.LikeCount != 1 {
		t.Fatalf("文章点赞数未共同提交: %d", article.LikeCount)
	}
}

// TestArticleLikeRollsBackWhenCounterFails 验证文章点赞数失败时关系回滚。
func TestArticleLikeRollsBackWhenCounterFails(t *testing.T) {
	db := openLikeTestDB(t)
	tx, _ := platformtransaction.NewManager(db)
	service := likeapp.NewService(
		NewArticleLikeRepository(db), NewCommentLikeRepository(db), nil, nil,
		failingProjection{}, tx,
	)

	if err := service.ArticleLike(context.Background(), 2, 1); err == nil {
		t.Fatal("点赞数更新失败时应返回错误")
	}
	var count int64
	if err := db.Model(&likemodel.ArticleLike{}).Count(&count).Error; err != nil {
		t.Fatalf("查询点赞关系失败: %v", err)
	}
	if count != 0 {
		t.Fatalf("点赞关系没有随事务回滚: %d", count)
	}
}
