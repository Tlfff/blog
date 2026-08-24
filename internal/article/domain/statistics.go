package domain

import "context"

// StatisticsWriter 定义 Article 上下文拥有的互动统计写入能力。
type StatisticsWriter interface {
	// IncrementCommentCount 按增量调整文章评论数。
	IncrementCommentCount(ctx context.Context, articleID uint64, delta int64) error
	// IncrementLikeCount 按增量调整文章点赞数。
	IncrementLikeCount(ctx context.Context, articleID uint64, delta int64) error
}
