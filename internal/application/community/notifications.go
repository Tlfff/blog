package community

import (
	domaincommunity "blog/internal/domain/community"
	notificationdto "blog/internal/dto/notification"
	"context"
	"time"
)

// 分页获取当前用户的通知列表
func (s *Service) GetMyNotifications(ctx context.Context, userID uint64, page, pageSize int64) (*notificationdto.NotificationListResponse, error) {
	// 1. 兜底分页参数，页码从 1 开始，单页上限 200 条
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize >= 200 {
		pageSize = 10
	}
	// 2. 查询通知列表
	list, err := s.notifications.GetList(ctx, userID, page, pageSize)
	if err != nil {
		return nil, err
	}
	// 3. 转换为响应 DTO
	return buildNotificationListResponse(list, page, pageSize), nil
}

// 把当前用户的全部未读通知标记为已读
func (s *Service) ClearUnread(ctx context.Context, userID uint64) error {
	return s.notifications.MarkAllAsRead(ctx, userID)
}

// 查询当前用户的未读通知数量
func (s *Service) GetUnreadCount(ctx context.Context, userID uint64) (int64, error) {
	return s.notifications.GetUnreadCount(ctx, userID)
}

// 根据点赞事件生成一条点赞文章通知
func (s *Service) CreateLikeNotification(ctx context.Context, event domaincommunity.NotificationEvent) error {
	// 1. 查询被点赞的文章，拿到作者ID作为接收者
	article, err := s.articles.FindByID(ctx, event.TargetID)
	if err != nil {
		return err
	}
	// 2. 查询点赞人的公开信息作为发送方
	user, err := s.users.FindUserByID(ctx, event.SenderID)
	if err != nil {
		return err
	}
	// 3. 组装并写入通知
	return s.sendNotification(ctx, domaincommunity.NotifyTypeLikeArticle, user, article.AuthorID, domaincommunity.LikeArticleContent{
		ArticleID:    article.ID,
		ArticleTitle: article.Title,
	})
}

// 组装并写入一条通知，发送者与接收者相同时不通知自己
func (s *Service) sendNotification(ctx context.Context, notifyType int8, sender *domaincommunity.UserInfo, receiverID uint64, content any) error {
	// 1. 自己操作自己的内容时不产生通知
	if sender.ID == receiverID {
		return nil
	}
	// 2. 组装通知领域对象，默认未读
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
	// 3. 写入通知存储
	return s.notifications.Insert(ctx, notification)
}

// 把领域通知列表转换为对外响应 DTO
func buildNotificationListResponse(list []*domaincommunity.Notification, page, pageSize int64) *notificationdto.NotificationListResponse {
	resp := &notificationdto.NotificationListResponse{
		List:     make([]*notificationdto.NotifyListItem, 0, len(list)),
		Page:     page,
		PageSize: pageSize,
	}
	// 1. 逐条转换通知的公共字段
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
		// 2. 按通知类型填充动作文案与业务内容
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
		// 3. 追加到结果列表
		resp.List = append(resp.List, item)
	}
	return resp
}

// 把 Mongo 反序列化出的 map 内容还原为点赞文章通知内容
func contentMap(v any) (domaincommunity.LikeArticleContent, bool) {
	// 1. 仅处理 map 结构，逐字段做类型断言
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

// 把 JSON/BSON 解出的任意数值类型统一转换为 uint64
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
