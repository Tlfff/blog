package oss

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// minioObjectClient 定义对象存储客户端所需的最小 MinIO 能力。
type minioObjectClient interface {
	PresignedPutObject(ctx context.Context, bucketName, objectName string, expires time.Duration) (*url.URL, error) // 生成预签名 PUT 地址
	RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error          // 删除单个对象
	ListObjects(ctx context.Context, bucketName string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo      // 按条件列出对象
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
	return m.client.RemoveObject(ctx, m.bucket, objectKey, minio.RemoveObjectOptions{})
}

// DeleteObjectsByPrefix 删除指定前缀下的全部对象。
func (m *MinioClient) DeleteObjectsByPrefix(ctx context.Context, prefix string) error {
	// 1. 拒绝空前缀，避免误删整个存储桶
	if strings.TrimSpace(prefix) == "" {
		return fmt.Errorf("object prefix is empty")
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("delete objects by prefix canceled: %w", err)
	}

	// 2. 使用可取消上下文递归列出前缀下的全部对象
	listCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	objects := m.client.ListObjects(listCtx, m.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	// 3. 逐个删除对象，失败时取消并排空列表通道，避免列表 goroutine 泄漏
	for object := range objects {
		if object.Err != nil {
			cancel()
			drainObjectInfos(objects)
			return fmt.Errorf("list objects by prefix %s failed: %w", prefix, object.Err)
		}
		if err := m.client.RemoveObject(listCtx, m.bucket, object.Key, minio.RemoveObjectOptions{}); err != nil {
			cancel()
			drainObjectInfos(objects)
			return fmt.Errorf("delete object %s failed: %w", object.Key, err)
		}
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("delete objects by prefix canceled: %w", err)
	}
	return nil
}

// drainObjectInfos 排空对象列表通道，确保 MinIO 列表 goroutine 正常退出。
func drainObjectInfos(objects <-chan minio.ObjectInfo) {
	// 1. 持续读取直到通道关闭
	for range objects {
	}
}

// GetObjectURL 获取对象完整访问 URL。
func (m *MinioClient) GetObjectURL(publicDomain, objectKey string) string {
	return publicDomain + "/" + objectKey
}
