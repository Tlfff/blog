package infrastructure

import (
	domain "blog/internal/article/domain"
	"blog/internal/shared/common"
	"blog/pkg/kafka"
	message "blog/shared/contracts/events/messages"
	"context"
)

type viewEventPublisher struct{ client *kafka.Client }

func NewViewEventPublisher(client *kafka.Client) domain.ViewEventPublisher {
	return &viewEventPublisher{client: client}
}

func (p *viewEventPublisher) SendViewHistory(ctx context.Context, event domain.ViewHistoryEvent) error {
	if p.client == nil || p.client.GetProducer() == nil {
		return common.ErrKafkaClientClosed
	}
	return p.client.GetProducer().SendViewHistory(ctx, &message.ViewHistoryMsg{ArticleID: event.ArticleID, UserID: event.UserID, CreatedTime: event.CreatedTime})
}
