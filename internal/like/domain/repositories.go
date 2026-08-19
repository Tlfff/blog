package domain

import "context"

type LikeTarget string

const (
	LikeTargetArticle  LikeTarget = "article"
	LikeTargetComment  LikeTarget = "comment"
	LikeStatusLiked    int8       = 1
	LikeStatusCanceled int8       = 2
)

type ArticleLikeRepository interface {
	SetLiked(ctx context.Context, userID, articleID uint64, liked bool) error
	IsLiked(ctx context.Context, userID, articleID uint64) (bool, error)
	GetLikedUserIDs(ctx context.Context, articleID uint64) ([]uint64, error)
}

type CommentLikeRepository interface {
	SetLiked(ctx context.Context, userID, commentID uint64, liked bool) error
	IsLiked(ctx context.Context, userID, commentID uint64) (bool, error)
	GetLikedUserIDs(ctx context.Context, commentID uint64) ([]uint64, error)
}

type LikeCache interface {
	IsLiked(ctx context.Context, target LikeTarget, targetID, userID uint64) (bool, error)
	Add(ctx context.Context, target LikeTarget, targetID, userID uint64) error
	Remove(ctx context.Context, target LikeTarget, targetID, userID uint64) error
}

type LikeCountStore interface {
	GetCommentLikeCounts(ctx context.Context, commentIDs []uint64) (map[uint64]uint64, error)
}

// ArticleLikeStatistics 更新文章点赞统计
type ArticleLikeStatistics interface {
	IncrementLikeCount(ctx context.Context, articleID uint64, delta int64) error
}

// CommentLikeStatistics 更新评论点赞统计
type CommentLikeStatistics interface {
	IncrementLikeCount(ctx context.Context, commentID uint64, delta int64) error
}
