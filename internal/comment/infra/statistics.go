package infra

import (
	domaincomment "blog/internal/comment/domain"
	"blog/internal/comment/infra/model"
	platformtransaction "blog/internal/platform/transaction"
	"context"

	"gorm.io/gorm"
)

// commentStatistics 是 Comment 上下文的点赞统计写入适配器
type commentStatistics struct{ db *gorm.DB }

// NewCommentLikeStatistics 创建评论点赞统计 Port 适配器。
func NewCommentLikeStatistics(db *gorm.DB) domaincomment.LikeCountProjection {
	return &commentStatistics{db: db}
}

// IncrementLikeCount 更新评论点赞数量。
func (s *commentStatistics) IncrementLikeCount(ctx context.Context, commentID uint64, delta int64) error {
	return platformtransaction.DB(ctx, s.db).WithContext(ctx).Model(&model.Comment{}).Where("id = ?", commentID).UpdateColumn("like_count", gorm.Expr("like_count + ?", delta)).Error
}
