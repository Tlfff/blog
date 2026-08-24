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
	SetLiked(ctx context.Context, userID, articleID uint64, liked bool) (bool, error)
	IsLiked(ctx context.Context, userID, articleID uint64) (bool, error)
	GetLikedUserIDs(ctx context.Context, articleID uint64) ([]uint64, error)
}

// CommentLikeRepository 定义评论点赞关系持久化能力。
type CommentLikeRepository interface {
	SetLiked(ctx context.Context, userID, commentID uint64, liked bool) (bool, error)
	IsLiked(ctx context.Context, userID, commentID uint64) (bool, error)
	GetLikedUserIDs(ctx context.Context, commentID uint64) ([]uint64, error)
}

// LikeCache 定义点赞状态缓存能力。
type LikeCache interface {
	IsLiked(ctx context.Context, target LikeTarget, targetID, userID uint64) (bool, error)
	Add(ctx context.Context, target LikeTarget, targetID, userID uint64) error
	Remove(ctx context.Context, target LikeTarget, targetID, userID uint64) error
}

// LikeCountStore 定义评论点赞数批量查询能力。
type LikeCountStore interface {
	GetCommentLikeCounts(ctx context.Context, commentIDs []uint64) (map[uint64]uint64, error)
}

// ArticleLikeStatistics 定义文章点赞数 Application Facade。
type ArticleLikeStatistics interface {
	AdjustLikeCount(ctx context.Context, articleID uint64, delta int64) error
}

// CommentLikeStatistics 定义评论点赞数 Application Facade。
type CommentLikeStatistics interface {
	AdjustLikeCount(ctx context.Context, commentID uint64, delta int64) error
}
