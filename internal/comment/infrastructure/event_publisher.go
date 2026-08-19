package infrastructure

import (
	domain "blog/internal/comment/domain"
	"blog/pkg/kafka"
	message "blog/shared/contracts/events/messages"
	"context"
)

// commentEventPublisher 是 Comment 上下文的 Kafka 事件发布适配器
type commentEventPublisher struct {
	client *kafka.Client // Kafka 客户端
}

// 创建评论事件发布适配器
func NewCommentEventPublisher(client *kafka.Client) domain.CommentEventPublisher {
	return &commentEventPublisher{client: client}
}

// 发布评论或回复创建事件
func (p *commentEventPublisher) PublishCommentCreated(_ context.Context, event domain.CommentCreated) error {
	if p.client == nil || p.client.GetProducer() == nil {
		return nil
	}
	notifyType := int8(3)
	targetID := event.ArticleID
	if event.RootID > 0 {
		notifyType = 4
		targetID = event.RootID
	}
	p.client.GetProducer().SendNotificationAsync(&message.NotificationMsg{EventID: event.EventID, Version: event.Version, NotifyType: notifyType, SenderID: event.UserID, TargetID: targetID, CreatedTime: event.CreatedTime})
	return nil
}
