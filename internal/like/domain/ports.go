package domain

import "context"

// TargetQuery 验证可点赞目标，但不暴露 Article 或 Comment 聚合。
type TargetQuery interface {
	ArticleExists(ctx context.Context, articleID uint64) (bool, error)
	CommentExists(ctx context.Context, commentID uint64) (bool, error)
}

// EventPublisher 发布 Like 上下文的集成事件。
type EventPublisher interface {
	PublishLikeCreated(ctx context.Context, event LikeCreated) error
	PublishLikeCanceled(ctx context.Context, event LikeCanceled) error
}
