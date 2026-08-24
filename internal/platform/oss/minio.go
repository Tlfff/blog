package oss

import (
	"context"
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinioClient 封装 MinIO Client 和默认存储桶。
type MinioClient struct {
	client *minio.Client // MinIO SDK 客户端
	bucket string        // 默认存储桶名称
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
	return m.client.RemoveObject(ctx, m.bucket, objectKey, minio.RemoveObjectOptions{})
}

// MoveObject 在同一存储桶中移动对象。
func (m *MinioClient) MoveObject(ctx context.Context, srcKey, dstKey string) error {
	_, err := m.client.CopyObject(ctx, minio.CopyDestOptions{
		Bucket: m.bucket,
		Object: dstKey,
	}, minio.CopySrcOptions{
		Bucket: m.bucket,
		Object: srcKey,
	})
	if err != nil {
		return fmt.Errorf("图片移动失败(%s -> %s): %w", srcKey, dstKey, err)
	}
	return m.client.RemoveObject(ctx, m.bucket, srcKey, minio.RemoveObjectOptions{})
}

// GetObjectURL 获取对象完整访问 URL。
func (m *MinioClient) GetObjectURL(publicDomain, objectKey string) string {
	return publicDomain + "/" + objectKey
}
