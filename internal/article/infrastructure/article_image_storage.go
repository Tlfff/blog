package infrastructure

import (
	domaincontent "blog/internal/article/domain"
	"blog/pkg/oss"
	"context"
	"time"
)

// articleImageStorageAdapter 是文章图片存储 Port 的 MinIO 适配器。
type articleImageStorageAdapter struct {
	oss *oss.MinioClient // MinIO 客户端
}

// NewArticleImageStorage 将 MinIO 客户端适配为文章图片存储 Port。
func NewArticleImageStorage(client *oss.MinioClient) domaincontent.ArticleImageStorage {
	return &articleImageStorageAdapter{oss: client}
}

// 生成文章图片上传的预签名 PUT 地址
func (a *articleImageStorageAdapter) PresignedPutURL(ctx context.Context, objectKey string, ttl time.Duration) (string, error) {
	return a.oss.PresignedPutURL(ctx, objectKey, ttl)
}

// 拼接图片对象的公开访问 URL
func (a *articleImageStorageAdapter) GetObjectURL(publicDomain, objectKey string) string {
	return a.oss.GetObjectURL(publicDomain, objectKey)
}

// 移动图片对象，用于把临时目录的图片转正为正式路径
func (a *articleImageStorageAdapter) MoveObject(ctx context.Context, srcKey, dstKey string) error {
	return a.oss.MoveObject(ctx, srcKey, dstKey)
}
