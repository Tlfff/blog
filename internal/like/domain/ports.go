package domain

import "context"

// EventPublisher 发布当前基线已接线的文章点赞通知消息。
type EventPublisher interface {
	// PublishLikeCreated 发布文章点赞创建消息。
	PublishLikeCreated(ctx context.Context, event LikeCreated) error
}

// ProjectionUpdater 更新文章或评论上下文拥有的点赞数。
type ProjectionUpdater interface {
	// ApplyLikeDelta 按增量调整目标点赞数。
	ApplyLikeDelta(ctx context.Context, target LikeTarget, targetID uint64, delta int64) error
}
