package community

import (
	domaincommunity "blog/internal/domain/community"
	"blog/internal/model"
	"blog/internal/repository"
	"context"
)

type notificationRepositoryAdapter struct {
	repo repository.NotificationRepository
}

// NewNotificationRepository 将 MongoDB 通知 Repository 适配为 Community 领域 Port。
func NewNotificationRepository(repo repository.NotificationRepository) domaincommunity.NotificationRepository {
	return &notificationRepositoryAdapter{repo: repo}
}

func (a *notificationRepositoryAdapter) Insert(ctx context.Context, notification *domaincommunity.Notification) error {
	m := toModelNotification(notification)
	if err := a.repo.Insert(ctx, m); err != nil {
		return err
	}
	notification.ID = m.ID.Hex()
	return nil
}

func (a *notificationRepositoryAdapter) GetList(ctx context.Context, receiverID uint64, page, pageSize int64) ([]*domaincommunity.Notification, error) {
	models, err := a.repo.GetList(ctx, receiverID, page, pageSize)
	if err != nil {
		return nil, err
	}
	list := make([]*domaincommunity.Notification, 0, len(models))
	for _, m := range models {
		list = append(list, toDomainNotification(m))
	}
	return list, nil
}

func (a *notificationRepositoryAdapter) MarkAllAsRead(ctx context.Context, receiverID uint64) error {
	return a.repo.MarkAllAsRead(ctx, receiverID)
}

func (a *notificationRepositoryAdapter) GetUnreadCount(ctx context.Context, receiverID uint64) (int64, error) {
	return a.repo.GetUnreadCount(ctx, receiverID)
}

func toModelNotification(n *domaincommunity.Notification) *model.Notification {
	sender := model.NotifySender{
		UserID:   n.Sender.UserID,
		Nickname: n.Sender.Nickname,
		Avatar:   n.Sender.Avatar,
	}
	var content any
	if like, ok := n.Content.(domaincommunity.LikeArticleContent); ok {
		content = model.LikeArticleNotifyContent{
			ArticleID:    like.ArticleID,
			ArticleTitle: like.ArticleTitle,
		}
	}
	return &model.Notification{
		ReceiverID:  n.ReceiverID,
		Sender:      sender,
		Type:        n.Type,
		IsRead:      n.IsRead,
		Content:     content,
		CreatedTime: n.CreatedTime,
	}
}

func toDomainNotification(m *model.Notification) *domaincommunity.Notification {
	n := &domaincommunity.Notification{
		ID:         m.ID.Hex(),
		ReceiverID: m.ReceiverID,
		Sender: domaincommunity.NotifySender{
			UserID:   m.Sender.UserID,
			Nickname: m.Sender.Nickname,
			Avatar:   m.Sender.Avatar,
		},
		Type:        m.Type,
		IsRead:      m.IsRead,
		CreatedTime: m.CreatedTime,
	}
	if content, ok := m.Content.(model.LikeArticleNotifyContent); ok {
		n.Content = domaincommunity.LikeArticleContent{
			ArticleID:    content.ArticleID,
			ArticleTitle: content.ArticleTitle,
		}
	}
	return n
}
