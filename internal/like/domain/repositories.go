package domain

import "context"

// LikeTarget 表示点赞目标类型。
type LikeTarget string

const (
	LikeTargetArticle  LikeTarget = "article" // 文章点赞目标
	LikeTargetComment  LikeTarget = "comment" // 评论点赞目标
	LikeStatusLiked    int8       = 1         // 点赞状态：已点赞
	LikeStatusCanceled int8       = 2         // 点赞状态：已取消
)

// ArticleLikeRepository 定义文章点赞关系持久化能力。
type ArticleLikeRepository interface {
	// SetLiked 更新用户文章点赞状态，并返回状态是否发生变化。
	SetLiked(ctx context.Context, userID, articleID uint64, liked bool) (bool, error)
	// IsLiked 查询用户是否已点赞文章。
	IsLiked(ctx context.Context, userID, articleID uint64) (bool, error)
	// GetLikedUserIDs 查询点赞文章的全部用户唯一标识。
	GetLikedUserIDs(ctx context.Context, articleID uint64) ([]uint64, error)
}

// CommentLikeRepository 定义评论点赞关系持久化能力。
type CommentLikeRepository interface {
	// SetLiked 更新用户评论点赞状态，并返回状态是否发生变化。
	SetLiked(ctx context.Context, userID, commentID uint64, liked bool) (bool, error)
	// IsLiked 查询用户是否已点赞评论。
	IsLiked(ctx context.Context, userID, commentID uint64) (bool, error)
	// GetLikedUserIDs 查询点赞评论的全部用户唯一标识。
	GetLikedUserIDs(ctx context.Context, commentID uint64) ([]uint64, error)
}

// LikeCache 定义点赞状态缓存能力。
type LikeCache interface {
	// IsLiked 查询用户对目标的点赞状态。
	IsLiked(ctx context.Context, target LikeTarget, targetID, userID uint64) (bool, error)
	// Add 将用户加入目标点赞缓存。
	Add(ctx context.Context, target LikeTarget, targetID, userID uint64) error
	// Remove 将用户移出目标点赞缓存。
	Remove(ctx context.Context, target LikeTarget, targetID, userID uint64) error
}

// LikeCountStore 定义评论点赞数批量查询能力。
type LikeCountStore interface {
	// GetCommentLikeCounts 批量查询评论点赞数。
	GetCommentLikeCounts(ctx context.Context, commentIDs []uint64) (map[uint64]uint64, error)
}

// ArticleLikeStatistics 定义文章点赞数 Application Facade。
type ArticleLikeStatistics interface {
	// AdjustLikeCount 按增量调整文章点赞数。
	AdjustLikeCount(ctx context.Context, articleID uint64, delta int64) error
}

// CommentLikeStatistics 定义评论点赞数 Application Facade。
type CommentLikeStatistics interface {
	// AdjustLikeCount 按增量调整评论点赞数。
	AdjustLikeCount(ctx context.Context, commentID uint64, delta int64) error
}
