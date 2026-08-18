// Package ports 定义领域层依赖的 Repository、外部资源与事件发布 Port。
package ports

import (
	"context"
	"time"
)

// Repository 是所有领域 Repository Port 的统一标记接口，具体模块在其上扩展方法。
type Repository interface{}

// ExternalResource 是所有外部资源 Port 的统一标记接口，例如对象存储、消息队列。
type ExternalResource interface{}

// EventPublisher 用于发布需要异步处理的领域事件。
type EventPublisher interface {
	Publish(ctx context.Context, event any) error
}

// UnitOfWork 抽象事务边界，Application 层通过它编排跨 Repository 的原子操作。
type UnitOfWork interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// ObjectStorage 抽象对象存储能力，供头像、文章图片等资源适配。
type ObjectStorage interface {
	PresignedPutURL(ctx context.Context, objectKey string, ttl time.Duration) (string, error)
	GetObjectURL(publicDomain, objectKey string) string
	DeleteObject(ctx context.Context, objectKey string) error
}
