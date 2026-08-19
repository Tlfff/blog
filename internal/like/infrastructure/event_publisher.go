package infrastructure

import (
	"blog/internal/like/domain"
	"blog/pkg/kafka"
	message "blog/shared/contracts/events/messages"
	"context"
)

type eventPublisher struct{ client *kafka.Client }

func NewEventPublisher(client *kafka.Client) domain.EventPublisher {
	return &eventPublisher{client: client}
}

func (p *eventPublisher) PublishLikeCreated(_ context.Context, event domain.LikeCreated) error {
	if p.client == nil || p.client.GetProducer() == nil {
		return nil
	}
	notifyType := int8(1)
	if event.Target == domain.LikeTargetComment {
		notifyType = 2
	}
	p.client.GetProducer().SendNotificationAsync(&message.NotificationMsg{EventID: event.EventID, Version: event.Version, NotifyType: notifyType, SenderID: event.UserID, TargetID: event.TargetID, CreatedTime: event.OccurredAt})
	return nil
}

func (p *eventPublisher) PublishLikeCanceled(context.Context, domain.LikeCanceled) error { return nil }
