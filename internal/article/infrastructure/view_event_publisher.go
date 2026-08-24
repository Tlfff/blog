package infrastructure

import (
	domain "blog/internal/article/domain"
	"blog/internal/platform/kafka"
	"blog/internal/platform/kafka/message"
	apperrors "blog/internal/shared/apperrors"
	"context"
)

type viewEventPublisher struct{ client *kafka.Client }

func NewViewEventPublisher(client *kafka.Client) domain.ViewEventPublisher {
	return &viewEventPublisher{client: client}
}

func (p *viewEventPublisher) SendViewHistory(ctx context.Context, event domain.ViewHistoryEvent) error {
	if p.client == nil || p.client.GetProducer() == nil {
		return apperrors.ErrKafkaClientClosed
	}
	return p.client.GetProducer().SendViewHistory(ctx, &message.ViewHistoryMsg{ArticleID: event.ArticleID, UserID: event.UserID, CreatedTime: event.CreatedTime})
}
