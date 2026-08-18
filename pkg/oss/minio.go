// Package oss 封装 MinIO 对象存储的预签名上传、移动与删除能力。
package oss

import (
	"context"
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinioClient 是 MinIO 对象存储客户端封装。
type MinioClient struct {
	client *minio.Client // MinIO 原生客户端
	bucket string        // 默认操作的 bucket 名称
}

// 创建 MinIO 客户端封装，useSSL 控制是否使用 HTTPS 访问
func NewMinioClient(endpoint, ak, sk, bucket string, useSSL bool) (*MinioClient, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(ak, sk, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client init failed: %w", err)
	}
	return &MinioClient{client: client, bucket: bucket}, nil
}

// 生成预签名 PUT URL，调用方可在 ttl 有效期内直传文件到对象存储
func (m *MinioClient) PresignedPutURL(ctx context.Context, objectKey string, ttl time.Duration) (string, error) {
	url, err := m.client.PresignedPutObject(ctx, m.bucket, objectKey, ttl)
	if err != nil {
		return "", fmt.Errorf("generate presigned put url failed: %w", err)
	}
	return url.String(), nil
}

// 删除指定对象
func (m *MinioClient) DeleteObject(ctx context.Context, objectKey string) error {
	return m.client.RemoveObject(ctx, m.bucket, objectKey, minio.RemoveObjectOptions{})
}

// 移动对象（同一 bucket 内）：先 Copy 到目标，成功后再删除源，避免拷贝失败丢数据
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

// 拼接对象的对外访问完整 URL
func (m *MinioClient) GetObjectURL(publicDomain, objectKey string) string {
	return publicDomain + "/" + objectKey
}
