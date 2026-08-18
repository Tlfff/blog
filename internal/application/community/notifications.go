package community

import (
	domaincommunity "blog/internal/domain/community"
	notificationdto "blog/internal/dto/notification"
	"context"
	"time"
)

func (s *Service) GetMyNotifications(ctx context.Context, userID uint64, page, pageSize int64) (*notificationdto.NotificationListResponse, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize >= 200 {
		pageSize = 10
	}
	list, err := s.notifications.GetList(ctx, userID, page, pageSize)
	if err != nil {
		return nil, err
	}
	return buildNotificationListResponse(list, page, pageSize), nil
}

func (s *Service) ClearUnread(ctx context.Context, userID uint64) error {
	return s.notifications.MarkAllAsRead(ctx, userID)
}

func (s *Service) GetUnreadCount(ctx context.Context, userID uint64) (int64, error) {
	return s.notifications.GetUnreadCount(ctx, userID)
}

func (s *Service) CreateLikeNotification(ctx context.Context, event domaincommunity.NotificationEvent) error {
	article, err := s.articles.FindByID(ctx, event.TargetID)
	if err != nil {
		return err
	}
	user, err := s.users.FindUserByID(ctx, event.SenderID)
	if err != nil {
		return err
	}
	return s.sendNotification(ctx, domaincommunity.NotifyTypeLikeArticle, user, article.AuthorID, domaincommunity.LikeArticleContent{
		ArticleID:    article.ID,
		ArticleTitle: article.Title,
	})
}

func (s *Service) sendNotification(ctx context.Context, notifyType int8, sender *domaincommunity.UserInfo, receiverID uint64, content any) error {
	if sender.ID == receiverID {
		return nil
	}
	notification := &domaincommunity.Notification{
		ReceiverID: receiverID,
		Sender: domaincommunity.NotifySender{
			UserID:   sender.ID,
			Nickname: sender.Nickname,
			Avatar:   sender.Avatar,
		},
		Type:        notifyType,
		IsRead:      false,
		Content:     content,
		CreatedTime: time.Now(),
	}
	return s.notifications.Insert(ctx, notification)
}

func buildNotificationListResponse(list []*domaincommunity.Notification, page, pageSize int64) *notificationdto.NotificationListResponse {
	resp := &notificationdto.NotificationListResponse{
		List:     make([]*notificationdto.NotifyListItem, 0, len(list)),
		Page:     page,
		PageSize: pageSize,
	}
	for _, m := range list {
		item := &notificationdto.NotifyListItem{
			ID:             m.ID,
			Type:           m.Type,
			IsRead:         m.IsRead,
			CreatedTime:    m.CreatedTime.Unix(),
			SenderID:       m.Sender.UserID,
			SenderNickname: m.Sender.Nickname,
			SenderAvatar:   m.Sender.Avatar,
		}
		switch m.Type {
		case domaincommunity.NotifyTypeLikeArticle:
			content, ok := m.Content.(domaincommunity.LikeArticleContent)
			if !ok {
				if like, ok := contentMap(m.Content); ok {
					content = like
				}
			}
			item.ActionText = "赞了你的文章"
			item.ArticleID = content.ArticleID
			item.Title = content.ArticleTitle
		default:
			item.ActionText = "收到一条新消息"
		}
		resp.List = append(resp.List, item)
	}
	return resp
}

func contentMap(v any) (domaincommunity.LikeArticleContent, bool) {
	if m, ok := v.(map[string]any); ok {
		articleID, idOK := asUint64(m["article_id"])
		articleTitle, titleOK := m["article_title"].(string)
		if idOK && titleOK {
			return domaincommunity.LikeArticleContent{
				ArticleID:    articleID,
				ArticleTitle: articleTitle,
			}, true
		}
	}
	return domaincommunity.LikeArticleContent{}, false
}

func asUint64(v any) (uint64, bool) {
	switch n := v.(type) {
	case float64:
		return uint64(n), true
	case int64:
		return uint64(n), true
	case int32:
		return uint64(n), true
	case uint64:
		return n, true
	default:
		return 0, false
	}
}
