package mq

import (
	domainnotification "blog/internal/notification/domain"
	"context"
	"time"

	"blog/pkg/kafka"
)

// NotificationConsumer 消费通知事件的接口。
type NotificationConsumer interface {
	CreateLikeNotification(ctx context.Context, event domainnotification.NotificationEvent) error
}

// ViewHistoryConsumer 消费浏览历史事件的接口。
type ViewHistoryConsumer interface {
	CreateViewHistory(ctx context.Context, userID, articleID uint64, timestamp time.Time) error
}

// 注册所有 Kafka 消息处理器
func RegisterHandlers(
	notificationConsumer NotificationConsumer,
	viewHistoryConsumer ViewHistoryConsumer,
) map[string]kafka.MessageHandler {
	return map[string]kafka.MessageHandler{
		"notification": NewNotificationHandler(notificationConsumer),
		"view_history": NewViewHistoryHandler(viewHistoryConsumer),
	}
}
