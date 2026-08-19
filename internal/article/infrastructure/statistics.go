package infrastructure

import (
	"blog/internal/article/infrastructure/model"
	commentdomain "blog/internal/comment/domain"
	likedomain "blog/internal/like/domain"
	"context"

	"gorm.io/gorm"
)

// articleStatistics 是 Article 上下文的互动统计写入适配器
type articleStatistics struct{ db *gorm.DB }

// 创建文章统计 Port 适配器
func NewArticleStatistics(db *gorm.DB) commentdomain.ArticleStatistics {
	return &articleStatistics{db: db}
}

// 创建文章点赞统计 Port 适配器
func NewArticleLikeStatistics(db *gorm.DB) likedomain.ArticleLikeStatistics {
	return &articleStatistics{db: db}
}

// 更新文章评论数量
func (s *articleStatistics) IncrementCommentCount(ctx context.Context, articleID uint64, delta int64) error {
	return s.db.WithContext(ctx).Model(&model.Article{}).Where("id = ?", articleID).UpdateColumn("comment_count", gorm.Expr("comment_count + ?", delta)).Error
}

// 更新文章点赞数量
func (s *articleStatistics) IncrementLikeCount(ctx context.Context, articleID uint64, delta int64) error {
	return s.db.WithContext(ctx).Model(&model.Article{}).Where("id = ?", articleID).UpdateColumn("like_count", gorm.Expr("like_count + ?", delta)).Error
}
