package domain

import (
	"context"
	"time"
)

type CommentRepository interface {
	CreateWithCounts(ctx context.Context, comment *Comment, incrementReply bool) error
	FindByID(ctx context.Context, id uint64) (*Comment, error)
	ListRootComments(ctx context.Context, articleID, lastID uint64, page, pageSize int, isDesc bool, authorID uint64) ([]*CommentWithUser, error)
	CountRootComments(ctx context.Context, articleID, authorID uint64) (int64, error)
	ListReplies(ctx context.Context, rootID, lastID uint64, page, pageSize int) ([]*CommentWithUser, error)
	CountReplies(ctx context.Context, rootID uint64) (int64, error)
	DeleteWithCounts(ctx context.Context, comment *Comment) error
}

type ArticleQuery interface {
	Exists(ctx context.Context, articleID uint64) (bool, error)
}

// ArticleStatistics 更新文章互动统计
type ArticleStatistics interface {
	IncrementCommentCount(ctx context.Context, articleID uint64, delta int64) error
}

type LikeCountQuery interface {
	GetCommentLikeCounts(ctx context.Context, commentIDs []uint64) (map[uint64]uint64, error)
}

// CommentEventPublisher 发布评论与回复事件
type CommentEventPublisher interface {
	PublishCommentCreated(ctx context.Context, event CommentCreated) error
}

// CommentCreated 是评论创建事件
type CommentCreated struct {
	EventID     string    // 事件唯一标识
	Version     int       // 事件契约版本
	UserID      uint64    // 评论作者用户ID
	ArticleID   uint64    // 评论所属文章ID
	RootID      uint64    // 根评论ID，主评论为0
	CreatedTime time.Time // 评论创建时间
}
