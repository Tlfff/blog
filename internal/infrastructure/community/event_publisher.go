package community

import (
	"blog/internal/common"
	domaincommunity "blog/internal/domain/community"
	"blog/internal/message"
	"blog/pkg/kafka"
	"context"
	"time"
)

type eventPublisherAdapter struct {
	kafkaClient *kafka.Client
}

// NewEventPublisher 将 Kafka 客户端适配为 Community 事件发布 Port。
func NewEventPublisher(kafkaClient *kafka.Client) domaincommunity.EventPublisher {
	return &eventPublisherAdapter{kafkaClient: kafkaClient}
}

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

func timeOrNow(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now()
	}
	return t
}
