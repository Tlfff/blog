package content

import (
	"context"
	"time"
)

// ArticleRepository 是文章持久化 Port。
type ArticleRepository interface {
	Create(ctx context.Context, article *Article) error
	FindByID(ctx context.Context, id uint64) (*Article, error)
	FindWithAuthorByID(ctx context.Context, id uint64) (*ArticleWithAuthor, error)
	Update(ctx context.Context, article *Article) error
	SoftDelete(ctx context.Context, articleID uint64) error
	Clear(ctx context.Context, articleID uint64) error
	ListWithCursor(ctx context.Context, lastID uint64, pageSize int, isDesc bool, status int8) ([]*Article, error)
	ListWithOffset(ctx context.Context, page, pageSize int, isDesc bool, status int8) ([]*Article, error)
	CountByStatus(ctx context.Context, status int8) (int64, error)
}

// ArticleImageStorage 是文章图片对象存储 Port。
type ArticleImageStorage interface {
	PresignedPutURL(ctx context.Context, objectKey string, ttl time.Duration) (string, error)
	GetObjectURL(publicDomain, objectKey string) string
	MoveObject(ctx context.Context, srcKey, dstKey string) error
}

// ArticleInteractionQuery 是文章互动统计查询 Port，由 Community 侧提供。
type ArticleInteractionQuery interface {
	IsUserLikedArticle(ctx context.Context, userID, articleID uint64) (bool, error)
}
