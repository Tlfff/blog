package infra

import (
	"blog/internal/like/domain"
	"blog/internal/platform/kafka"
	"blog/internal/platform/kafka/message"
	"context"
)

type eventPublisher struct {
	client *kafka.Client // Kafka 客户端
}

// NewEventPublisher 创建文章点赞通知发布器。
func NewEventPublisher(client *kafka.Client) domain.EventPublisher {
	return &eventPublisher{client: client}
}

// PublishLikeCreated 异步发布当前基线的文章点赞通知消息。
func (p *eventPublisher) PublishLikeCreated(_ context.Context, event domain.LikeCreated) error {
	if event.Target != domain.LikeTargetArticle || p.client == nil || p.client.GetProducer() == nil {
		return nil
	}
	p.client.GetProducer().SendNotificationAsync(&message.NotificationMsg{
		NotifyType:  1,
		SenderID:    event.UserID,
		TargetID:    event.TargetID,
		CreatedTime: event.OccurredAt,
	})
	return nil
}
