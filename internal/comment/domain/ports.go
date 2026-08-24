package domain

import "context"

// CommentRepository 定义评论持久化和只读查询能力。
type CommentRepository interface {
	// CreateWithCounts 创建评论，并使用当前事务维护根评论和文章计数。
	CreateWithCounts(ctx context.Context, comment *Comment, incrementReply bool) error
	// FindByID 根据唯一标识查询评论。
	FindByID(ctx context.Context, id uint64) (*Comment, error)
	// ListRootComments 查询文章主评论列表及展示用户信息。
	ListRootComments(ctx context.Context, articleID, lastID uint64, page, pageSize int, isDesc bool, authorID uint64) ([]*CommentWithUser, error)
	// CountRootComments 查询文章主评论数量。
	CountRootComments(ctx context.Context, articleID, authorID uint64) (int64, error)
	// ListReplies 查询根评论下的回复列表。
	ListReplies(ctx context.Context, rootID, lastID uint64, page, pageSize int) ([]*CommentWithUser, error)
	// CountReplies 查询根评论下的回复数量。
	CountReplies(ctx context.Context, rootID uint64) (int64, error)
	// DeleteWithCounts 删除评论，并使用当前事务维护根评论和文章计数。
	DeleteWithCounts(ctx context.Context, comment *Comment) error
}

// ArticleStatistics 定义 Comment 修改 Article 评论数所需的最小能力。
type ArticleStatistics interface {
	// IncrementCommentCount 按增量调整文章评论数。
	IncrementCommentCount(ctx context.Context, articleID uint64, delta int64) error
}

// LikeCountQuery 批量查询评论点赞数。
type LikeCountQuery interface {
	// GetCommentLikeCounts 查询指定评论的点赞数。
	GetCommentLikeCounts(ctx context.Context, commentIDs []uint64) (map[uint64]uint64, error)
}

// LikeCountProjection 更新 Comment 上下文拥有的点赞数。
type LikeCountProjection interface {
	// IncrementLikeCount 按增量调整评论点赞数。
	IncrementLikeCount(ctx context.Context, commentID uint64, delta int64) error
}
