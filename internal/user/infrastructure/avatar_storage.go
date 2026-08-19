package infrastructure

import (
	"blog/internal/shared/common"
	domainidentity "blog/internal/user/domain"
	"blog/pkg/oss"
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

// 生成头像上传的预签名 PUT 地址，客户端凭该地址直传对象存储
func (a *avatarStorageAdapter) PresignedPutURL(ctx context.Context, objectKey string, ttl time.Duration) (string, error) {
	if a.oss == nil {
		return "", common.ErrSystem
	}
	return a.oss.PresignedPutURL(ctx, objectKey, ttl)
}

// 拼接头像对象的公开访问 URL
func (a *avatarStorageAdapter) GetObjectURL(publicDomain, objectKey string) string {
	if a.oss == nil {
		return ""
	}
	return a.oss.GetObjectURL(publicDomain, objectKey)
}

// 删除头像对象，用于替换头像后清理旧文件
func (a *avatarStorageAdapter) DeleteObject(ctx context.Context, objectKey string) error {
	if a.oss == nil {
		return common.ErrSystem
	}
	return a.oss.DeleteObject(ctx, objectKey)
}
