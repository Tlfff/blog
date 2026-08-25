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

// fakeMinioObjectClient 是对象前缀删除测试使用的 MinIO 客户端。
type fakeMinioObjectClient struct {
	objects       []minio.ObjectInfo // 列表接口返回的对象
	listErr       error              // 列表接口返回的错误
	removeErr     error              // 删除对象时返回的错误
	removed       []string           // 已删除的对象 Key
	listCallCount int                // 列表接口调用次数
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

// ListObjects 返回测试对象列表或列表错误。
func (f *fakeMinioObjectClient) ListObjects(_ context.Context, _ string, _ minio.ListObjectsOptions) <-chan minio.ObjectInfo {
	// 1. 记录调用并准备完整缓冲通道，避免测试生产者阻塞
	f.listCallCount++
	itemCount := len(f.objects)
	if f.listErr != nil {
		itemCount++
	}
	objects := make(chan minio.ObjectInfo, itemCount)

	// 2. 写入测试对象及可选错误后关闭通道
	for _, object := range f.objects {
		objects <- object
	}
	if f.listErr != nil {
		objects <- minio.ObjectInfo{Err: f.listErr}
	}
	close(objects)
	return objects
}

// TestMinioClientDeleteObjectsByPrefix 验证前缀对象删除和空目录行为。
func TestMinioClientDeleteObjectsByPrefix(t *testing.T) {
	// 1. 验证递归列出的对象被逐个删除
	client := &fakeMinioObjectClient{objects: []minio.ObjectInfo{
		{Key: "article/7/a.png"},
		{Key: "article/7/b.jpg"},
	}}
	storage := &MinioClient{client: client, bucket: "blog"}

	if err := storage.DeleteObjectsByPrefix(context.Background(), "article/7/"); err != nil {
		t.Fatalf("按前缀删除对象失败: %v", err)
	}
	if !reflect.DeepEqual(client.removed, []string{"article/7/a.png", "article/7/b.jpg"}) {
		t.Fatalf("删除对象列表错误: %v", client.removed)
	}

	// 2. 验证空目录按成功处理
	emptyClient := &fakeMinioObjectClient{}
	emptyStorage := &MinioClient{client: emptyClient, bucket: "blog"}
	if err := emptyStorage.DeleteObjectsByPrefix(context.Background(), "article/8/"); err != nil {
		t.Fatalf("空目录应视为删除成功: %v", err)
	}
}

// TestMinioClientDeleteObjectsByPrefixErrors 验证列表、删除和上下文错误路径。
func TestMinioClientDeleteObjectsByPrefixErrors(t *testing.T) {
	// 1. 验证对象列表错误保留原始错误链
	listErr := errors.New("list failed")
	listClient := &fakeMinioObjectClient{listErr: listErr}
	if err := (&MinioClient{client: listClient, bucket: "blog"}).DeleteObjectsByPrefix(context.Background(), "article/7/"); !errors.Is(err, listErr) {
		t.Fatalf("列表错误未保留错误链: %v", err)
	}

	// 2. 验证单对象删除错误保留原始错误链
	removeErr := errors.New("remove failed")
	removeClient := &fakeMinioObjectClient{
		objects:   []minio.ObjectInfo{{Key: "article/7/a.png"}, {Key: "article/7/b.png"}},
		removeErr: removeErr,
	}
	if err := (&MinioClient{client: removeClient, bucket: "blog"}).DeleteObjectsByPrefix(context.Background(), "article/7/"); !errors.Is(err, removeErr) {
		t.Fatalf("删除错误未保留错误链: %v", err)
	}

	// 3. 验证已取消上下文不会启动对象列表请求
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	canceledClient := &fakeMinioObjectClient{}
	if err := (&MinioClient{client: canceledClient, bucket: "blog"}).DeleteObjectsByPrefix(canceledCtx, "article/7/"); !errors.Is(err, context.Canceled) {
		t.Fatalf("上下文取消错误不正确: %v", err)
	}
	if canceledClient.listCallCount != 0 {
		t.Fatalf("上下文已取消时不应请求对象列表: %d", canceledClient.listCallCount)
	}

	// 4. 验证空前缀被拒绝以避免误删存储桶
	emptyPrefixClient := &fakeMinioObjectClient{}
	if err := (&MinioClient{client: emptyPrefixClient, bucket: "blog"}).DeleteObjectsByPrefix(context.Background(), " "); err == nil {
		t.Fatal("空前缀应被拒绝")
	}
}
