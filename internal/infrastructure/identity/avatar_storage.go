package identity

import (
	"blog/internal/common"
	domainidentity "blog/internal/domain/identity"
	"blog/pkg/oss"
	"context"
	"time"
)

type avatarStorageAdapter struct {
	oss *oss.MinioClient
}

// NewAvatarStorage 将 MinIO 客户端适配为 Identity 头像存储 Port。
func NewAvatarStorage(client *oss.MinioClient) domainidentity.AvatarObjectStorage {
	return &avatarStorageAdapter{oss: client}
}

func (a *avatarStorageAdapter) PresignedPutURL(ctx context.Context, objectKey string, ttl time.Duration) (string, error) {
	if a.oss == nil {
		return "", common.ErrSystem
	}
	return a.oss.PresignedPutURL(ctx, objectKey, ttl)
}

func (a *avatarStorageAdapter) GetObjectURL(publicDomain, objectKey string) string {
	if a.oss == nil {
		return ""
	}
	return a.oss.GetObjectURL(publicDomain, objectKey)
}

func (a *avatarStorageAdapter) DeleteObject(ctx context.Context, objectKey string) error {
	if a.oss == nil {
		return common.ErrSystem
	}
	return a.oss.DeleteObject(ctx, objectKey)
}
