package infra

import (
	domaincontent "blog/internal/article/domain"
	"blog/internal/platform/oss"
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

// PresignedPutURL 生成文章图片上传的预签名 PUT 地址。
func (a *articleImageStorageAdapter) PresignedPutURL(ctx context.Context, objectKey string, ttl time.Duration) (string, error) {
	return a.oss.PresignedPutURL(ctx, objectKey, ttl)
}

// GetObjectURL 拼接图片对象的公开访问 URL。
func (a *articleImageStorageAdapter) GetObjectURL(publicDomain, objectKey string) string {
	return a.oss.GetObjectURL(publicDomain, objectKey)
}

// DeleteObjectsByPrefix 删除指定对象前缀下的全部文章图片。
func (a *articleImageStorageAdapter) DeleteObjectsByPrefix(ctx context.Context, prefix string) error {
	// 1. 委托平台对象存储删除指定前缀下的全部对象
	return a.oss.DeleteObjectsByPrefix(ctx, prefix)
}
