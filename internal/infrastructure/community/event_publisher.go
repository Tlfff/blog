package community

import (
	"blog/internal/common"
	domaincommunity "blog/internal/domain/community"
	"blog/internal/message"
	"blog/pkg/kafka"
	"context"
	"time"
)

// eventPublisherAdapter 是 Community 事件发布 Port 的 Kafka 实现。
type eventPublisherAdapter struct {
	kafkaClient *kafka.Client // Kafka 客户端，为 nil 表示未启用消息队列
}

// NewEventPublisher 将 Kafka 客户端适配为 Community 事件发布 Port。
func NewEventPublisher(kafkaClient *kafka.Client) domaincommunity.EventPublisher {
	return &eventPublisherAdapter{kafkaClient: kafkaClient}
}

// 异步发送点赞通知事件，Kafka 不可用时降级为静默丢弃（通知类消息允许丢失）
func (a *eventPublisherAdapter) SendLikeNotification(_ context.Context, event domaincommunity.NotificationEvent) error {
	if a.kafkaClient == nil {
		return nil
	}
	producer := a.kafkaClient.GetProducer()
	if producer == nil {
		return nil
	}
	producer.SendNotificationAsync(&message.NotificationMsg{
		NotifyType:  event.NotifyType,
		SenderID:    event.SenderID,
		TargetID:    event.TargetID,
		CreatedTime: event.CreatedTime,
	})
	return nil
}

// 同步发送浏览历史事件，Kafka 不可用时返回错误交由上层处理
func (a *eventPublisherAdapter) SendViewHistory(ctx context.Context, event domaincommunity.ViewHistoryEvent) error {
	if a.kafkaClient == nil {
		return common.ErrKafkaClientClosed
	}
	producer := a.kafkaClient.GetProducer()
	if producer == nil {
		return common.ErrKafkaClientClosed
	}
	return producer.SendViewHistory(ctx, &message.ViewHistoryMsg{
		ArticleID:   event.ArticleID,
		UserID:      event.UserID,
		CreatedTime: timeOrNow(event.CreatedTime),
	})
}

// 时间为零值时回退为当前时间，保证事件一定带有发生时间
func timeOrNow(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now()
	}
	return t
}
