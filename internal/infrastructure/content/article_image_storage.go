package content

import (
	domaincontent "blog/internal/domain/content"
	"blog/pkg/oss"
	"context"
	"time"
)

type articleImageStorageAdapter struct {
	oss *oss.MinioClient
}

// NewArticleImageStorage 将 MinIO 客户端适配为文章图片存储 Port。
func NewArticleImageStorage(client *oss.MinioClient) domaincontent.ArticleImageStorage {
	return &articleImageStorageAdapter{oss: client}
}

func (a *articleImageStorageAdapter) PresignedPutURL(ctx context.Context, objectKey string, ttl time.Duration) (string, error) {
	return a.oss.PresignedPutURL(ctx, objectKey, ttl)
}

func (a *articleImageStorageAdapter) GetObjectURL(publicDomain, objectKey string) string {
	return a.oss.GetObjectURL(publicDomain, objectKey)
}

func (a *articleImageStorageAdapter) MoveObject(ctx context.Context, srcKey, dstKey string) error {
	return a.oss.MoveObject(ctx, srcKey, dstKey)
}
