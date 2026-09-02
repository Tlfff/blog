package oss

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// minioObjectClient 定义对象存储客户端所需的最小 MinIO 能力。
type minioObjectClient interface {
	PresignedPutObject(ctx context.Context, bucketName, objectName string, expires time.Duration) (*url.URL, error) // 生成预签名 PUT 地址
	RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error          // 删除单个对象
}

// MinioClient 封装 MinIO Client 和默认存储桶。
type MinioClient struct {
	client minioObjectClient // MinIO SDK 客户端
	bucket string            // 默认存储桶名称
}

// NewMinioClient 创建 MinIO 客户端。
//
// 参数说明：
//   - endpoint：MinIO 服务地址。
//   - accessKeyID：访问密钥 ID。
//   - secretAccessKey：访问密钥。
//   - bucket：对象存储桶名称。
//   - useSSL：是否使用 HTTPS。
func NewMinioClient(endpoint, accessKeyID, secretAccessKey, bucket string, useSSL bool) (*MinioClient, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client init failed: %w", err)
	}
	return &MinioClient{client: client, bucket: bucket}, nil
}

// PresignedPutURL 生成预签名 PUT URL。
func (m *MinioClient) PresignedPutURL(ctx context.Context, objectKey string, ttl time.Duration) (string, error) {
	url, err := m.client.PresignedPutObject(ctx, m.bucket, objectKey, ttl)
	if err != nil {
		return "", fmt.Errorf("generate presigned put url failed: %w", err)
	}
	return url.String(), nil
}

// DeleteObject 删除对象。
func (m *MinioClient) DeleteObject(ctx context.Context, objectKey string) error {
	// 1. 删除指定对象并保留原始错误链
	if err := m.client.RemoveObject(ctx, m.bucket, objectKey, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("delete object %s failed: %w", objectKey, err)
	}
	return nil
}

// GetObjectURL 获取对象完整访问 URL。
func (m *MinioClient) GetObjectURL(publicDomain, objectKey string) string {
	return publicDomain + "/" + objectKey
}
