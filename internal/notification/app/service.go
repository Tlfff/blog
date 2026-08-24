package app

import (
	notificationdto "blog/internal/notification/app/dto"
	"blog/internal/notification/domain"
	"context"
)

// Service 编排通知生成、查询和已读状态用例。
type Service struct {
	notifications domain.NotificationRepository // 通知持久化 Port
	articles      domain.ArticleQuery           // 文章查询 Port
	users         domain.UserInfoQuery          // 用户查询 Port
}

// NewService 创建 Notification 上下文应用服务。
func NewService(notifications domain.NotificationRepository, articles domain.ArticleQuery, users domain.UserInfoQuery) *Service {
	return &Service{notifications: notifications, articles: articles, users: users}
}

// CreateLikeNotification 处理当前基线已接线的文章点赞通知。
func (s *Service) CreateLikeNotification(ctx context.Context, event domain.NotificationEvent) error {
	// 1. 由通知类型值对象校验类型集合和当前可创建类型
	notifyType, err := domain.NewNotificationType(event.NotifyType)
	if err != nil {
		return err
	}
	if err := notifyType.EnsureCreatable(); err != nil {
		return err
	}

	// 2. 查询发送方和文章快照
	sender, err := s.users.FindUserByID(ctx, event.SenderID)
	if err != nil {
		return err
	}
	article, err := s.articles.FindByID(ctx, event.TargetID)
	if err != nil {
		return err
	}

	// 3. 通过领域工厂创建通知，自通知场景直接结束
	notification, shouldCreate, err := domain.NewArticleLikeNotification(notifyType, *sender, *article, event.CreatedTime)
	if err != nil {
		return err
	}
	if !shouldCreate {
		return nil
	}

	// 4. 保存领域工厂创建的通知快照
	return s.notifications.Insert(ctx, notification)
}

// GetMyNotifications 分页查询当前用户通知列表。
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
	return notificationdto.NewNotificationListResponse(list, page, pageSize), nil
}

// ClearUnread 将当前用户全部未读通知标记为已读。
func (s *Service) ClearUnread(ctx context.Context, userID uint64) error {
	return s.notifications.MarkAllAsRead(ctx, userID)
}

// GetUnreadCount 查询当前用户未读通知数量。
func (s *Service) GetUnreadCount(ctx context.Context, userID uint64) (int64, error) {
	return s.notifications.GetUnreadCount(ctx, userID)
}
