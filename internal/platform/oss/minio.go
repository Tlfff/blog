package oss

import (
	"context"
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioClient struct {
	client *minio.Client
	bucket string
}

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

// 生成预签名 PUT URL
func (m *MinioClient) PresignedPutURL(ctx context.Context, objectKey string, ttl time.Duration) (string, error) {
	url, err := m.client.PresignedPutObject(ctx, m.bucket, objectKey, ttl)
	if err != nil {
		return "", fmt.Errorf("generate presigned put url failed: %w", err)
	}
	return url.String(), nil
}

// 删除对象
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

// 获取对象完整 URL
func (m *MinioClient) GetObjectURL(publicDomain, objectKey string) string {
	return publicDomain + "/" + objectKey
}
