package infra

import (
	"blog/internal/platform/oss"
	apperrors "blog/internal/shared/apperrors"
	domainidentity "blog/internal/user/domain"
	"context"
	"time"
)

// avatarStorageAdapter 是头像对象存储 Port 的 MinIO 适配器。
type avatarStorageAdapter struct {
	oss *oss.MinioClient // MinIO 客户端，为空表示未配置对象存储
}

// NewAvatarStorage 将 MinIO 客户端适配为 Identity 头像存储 Port。
func NewAvatarStorage(client *oss.MinioClient) domainidentity.AvatarObjectStorage {
	return &avatarStorageAdapter{oss: client}
}

// PresignedPutURL 生成头像上传的预签名 PUT 地址。
func (a *avatarStorageAdapter) PresignedPutURL(ctx context.Context, objectKey string, ttl time.Duration) (string, error) {
	if a.oss == nil {
		return "", apperrors.ErrSystem
	}
	return a.oss.PresignedPutURL(ctx, objectKey, ttl)
}

// GetObjectURL 拼接头像对象的公开访问 URL。
func (a *avatarStorageAdapter) GetObjectURL(publicDomain, objectKey string) string {
	if a.oss == nil {
		return ""
	}
	return a.oss.GetObjectURL(publicDomain, objectKey)
}

// DeleteObject 删除头像对象。
func (a *avatarStorageAdapter) DeleteObject(ctx context.Context, objectKey string) error {
	if a.oss == nil {
		return apperrors.ErrSystem
	}
	return a.oss.DeleteObject(ctx, objectKey)
}
