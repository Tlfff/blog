package oss

import (
	"context"
	"errors"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
)

// fakeMinioObjectClient 是对象存储测试使用的 MinIO 客户端。
type fakeMinioObjectClient struct {
	removeErr error    // 删除对象时返回的错误
	removed   []string // 已删除的对象 Key
}

// PresignedPutObject 返回测试预签名地址。
func (f *fakeMinioObjectClient) PresignedPutObject(context.Context, string, string, time.Duration) (*url.URL, error) {
	// 1. 返回固定测试地址
	return url.Parse("https://upload.example")
}

// RemoveObject 记录删除对象操作。
func (f *fakeMinioObjectClient) RemoveObject(_ context.Context, _ string, objectName string, _ minio.RemoveObjectOptions) error {
	// 1. 返回预设错误或记录删除对象
	if f.removeErr != nil {
		return f.removeErr
	}
	f.removed = append(f.removed, objectName)
	return nil
}

// TestMinioClientDeleteObject 验证单对象删除成功和错误链。
func TestMinioClientDeleteObject(t *testing.T) {
	// 1. 验证指定对象 Key 被传递给 MinIO 客户端
	client := &fakeMinioObjectClient{}
	storage := &MinioClient{client: client, bucket: "blog"}
	if err := storage.DeleteObject(context.Background(), "article/img/2026/08/a.png"); err != nil {
		t.Fatalf("删除对象失败: %v", err)
	}
	if !reflect.DeepEqual(client.removed, []string{"article/img/2026/08/a.png"}) {
		t.Fatalf("删除对象列表错误: %v", client.removed)
	}

	// 2. 验证删除错误保留原始错误链
	removeErr := errors.New("remove failed")
	errorStorage := &MinioClient{client: &fakeMinioObjectClient{removeErr: removeErr}, bucket: "blog"}
	if err := errorStorage.DeleteObject(context.Background(), "article/img/2026/08/a.png"); !errors.Is(err, removeErr) {
		t.Fatalf("删除错误未保留错误链: %v", err)
	}
}
