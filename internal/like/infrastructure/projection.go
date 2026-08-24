package infrastructure

import (
	"blog/internal/like/domain"
	"context"
)

type projectionUpdater struct {
	articles domain.ArticleLikeStatistics // Article Application Facade
	comments domain.CommentLikeStatistics // Comment Application Facade
}

// NewProjectionUpdater 创建点赞统计 Application Facade 适配器。
func NewProjectionUpdater(articles domain.ArticleLikeStatistics, comments domain.CommentLikeStatistics) domain.ProjectionUpdater {
	return &projectionUpdater{articles: articles, comments: comments}
}

// ApplyLikeDelta 按目标类型调整点赞数。
func (p *projectionUpdater) ApplyLikeDelta(ctx context.Context, target domain.LikeTarget, targetID uint64, delta int64) error {
	if target == domain.LikeTargetArticle {
		return p.articles.AdjustLikeCount(ctx, targetID, delta)
	}
	return p.comments.AdjustLikeCount(ctx, targetID, delta)
}
