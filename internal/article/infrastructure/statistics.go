package infrastructure

import (
	articledomain "blog/internal/article/domain"
	"blog/internal/article/infrastructure/model"
	platformtransaction "blog/internal/platform/transaction"
	"context"

	"gorm.io/gorm"
)

type articleStatistics struct {
	db *gorm.DB // 默认 GORM 数据库连接
}

// NewArticleStatistics 创建 Article 统计写入 Adapter。
func NewArticleStatistics(db *gorm.DB) articledomain.StatisticsWriter {
	return &articleStatistics{db: db}
}

// IncrementCommentCount 按增量调整文章评论数。
func (s *articleStatistics) IncrementCommentCount(ctx context.Context, articleID uint64, delta int64) error {
	return platformtransaction.DB(ctx, s.db).WithContext(ctx).Model(&model.Article{}).
		Where("id = ?", articleID).
		UpdateColumn("comment_count", gorm.Expr("comment_count + ?", delta)).Error
}

// IncrementLikeCount 按增量调整文章点赞数。
func (s *articleStatistics) IncrementLikeCount(ctx context.Context, articleID uint64, delta int64) error {
	return platformtransaction.DB(ctx, s.db).WithContext(ctx).Model(&model.Article{}).
		Where("id = ?", articleID).
		UpdateColumn("like_count", gorm.Expr("like_count + ?", delta)).Error
}
