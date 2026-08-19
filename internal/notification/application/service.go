// Package application 编排 Notification 上下文用例。
package application

import (
	domain "blog/internal/notification/domain"
	"context"
	"errors"
)

// Service 编排通知生成、查询和已读状态用例
type Service struct {
	notifications domain.NotificationRepository // 通知持久化 Port
	articles      domain.ArticleQuery           // 文章查询 Port
	users         domain.UserInfoQuery          // 用户查询 Port
	comments      domain.CommentQuery           // 评论查询 Port
}

// 创建 Notification 上下文应用服务
func NewService(notifications domain.NotificationRepository, articles domain.ArticleQuery, users domain.UserInfoQuery, comments domain.CommentQuery) *Service {
	return &Service{notifications: notifications, articles: articles, users: users, comments: comments}
}

// 根据业务事件创建通知并保持事件幂等
func (s *Service) CreateLikeNotification(ctx context.Context, event domain.NotificationEvent) error {
	sender, err := s.users.FindUserByID(ctx, event.SenderID)
	if err != nil {
		return err
	}
	receiverID, content, err := s.resolveTarget(ctx, event)
	if err != nil {
		return err
	}
	if sender.ID == receiverID {
		return nil
	}
	return s.notifications.Insert(ctx, &domain.Notification{
		EventID:     event.EventID,
		ReceiverID:  receiverID,
		Sender:      domain.NotifySender{UserID: sender.ID, Nickname: sender.Nickname, Avatar: sender.Avatar},
		Type:        event.NotifyType,
		Content:     content,
		CreatedTime: event.CreatedTime,
	})
}

// 解析通知目标并返回接收者与内容快照
func (s *Service) resolveTarget(ctx context.Context, event domain.NotificationEvent) (uint64, any, error) {
	switch event.NotifyType {
	case domain.NotifyTypeLikeArticle, domain.NotifyTypeCommentArticle:
		article, err := s.articles.FindByID(ctx, event.TargetID)
		if err != nil {
			return 0, nil, err
		}
		if event.NotifyType == domain.NotifyTypeLikeArticle {
			return article.AuthorID, domain.LikeArticleContent{ArticleID: article.ID, ArticleTitle: article.Title}, nil
		}
		return article.AuthorID, domain.CommentContent{ArticleID: article.ID}, nil
	case domain.NotifyTypeLikeComment, domain.NotifyTypeReplyComment:
		comment, err := s.comments.FindByID(ctx, event.TargetID)
		if err != nil {
			return 0, nil, err
		}
		return comment.UserID, domain.CommentContent{ArticleID: comment.ArticleID, CommentID: comment.ID}, nil
	default:
		return 0, nil, errors.New("不支持的通知类型")
	}
}

// 分页查询当前用户通知列表
func (s *Service) GetMyNotifications(ctx context.Context, userID uint64, page, pageSize int64) ([]*domain.Notification, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize >= 200 {
		pageSize = 10
	}
	return s.notifications.GetList(ctx, userID, page, pageSize)
}

// 将当前用户全部未读通知标记为已读
func (s *Service) ClearUnread(ctx context.Context, userID uint64) error {
	return s.notifications.MarkAllAsRead(ctx, userID)
}

// 查询当前用户未读通知数量
func (s *Service) GetUnreadCount(ctx context.Context, userID uint64) (int64, error) {
	return s.notifications.GetUnreadCount(ctx, userID)
}
