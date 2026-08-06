package service

import (
	// 假设你在这里定义了 DTO
	"blog/internal/dto/notification"
	"blog/internal/model"
	"blog/internal/repository"
	"context"
	"time"
)

type NotificationService struct {
	notifyRepo repository.NotificationRepository
}

func NewNotificationService(notifyRepo repository.NotificationRepository) *NotificationService {
	return &NotificationService{
		notifyRepo: notifyRepo,
	}
}

// 获取我的通知列表（传统分页）
func (s *NotificationService) GetMyNotifications(ctx context.Context, userID uint64, page, pageSize int64) (*notification.NotificationListResponse, error) {
	// 1. 业务层做基础的分页防御
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize >= 200 { // 限制单页最大数量，防止前端恶意拉取大物理块
		pageSize = 10
	}

	// 2. 获取通知数据
	list, err := s.notifyRepo.GetList(ctx, userID, page, pageSize)
	if err != nil {
		return nil, err
	}
	// 3. todo 获取通知总数

	// 4. 封装返回体
	rep := notification.NewNotificationListResponse(list, int64(page), int64(pageSize))
	return rep, nil
}

// 一键清除未读红点（全量已读）
func (s *NotificationService) ClearUnread(ctx context.Context, userID uint64) error {
	return s.notifyRepo.MarkAllAsRead(ctx, userID)
}

// 获取未读红点数量
func (s *NotificationService) GetUnreadCount(ctx context.Context, userID uint64) (int64, error) {
	return s.notifyRepo.GetUnreadCount(ctx, userID)
}

func (s *NotificationService) Insert(ctx context.Context, userID uint64, notify *model.Notification) error {
	// 如果点赞用户是自己，则不创建通知
	if notify.Sender.UserID == userID {
		return nil
	}
	notify.IsRead = false
	return s.notifyRepo.Insert(ctx, notify)
}

// 底层发送通知的统一入口
func (s *NotificationService) send(ctx context.Context, notifyType int8, senderID uint64, senderNickname, senderAvatar string, receiverID uint64, content any) error {
	// 1. 自己触发的操作（如自己赞自己、自己回复自己），不产生通知
	if senderID == receiverID {
		return nil
	}

	// 2. 统一组装基础的 Notification 文档结构
	notify := &model.Notification{
		ReceiverID:  receiverID,
		Type:        notifyType,
		IsRead:      false,
		CreatedTime: time.Now(),
		Sender: model.NotifySender{
			UserID:   senderID,
			Nickname: senderNickname,
			Avatar:   senderAvatar,
		},
		Content: content,
	}

	// 统一调用 Repository 入库
	return s.notifyRepo.Insert(ctx, notify)
}

// 对外：点赞文章通知
func (s *NotificationService) SendLikeArticleNotification(ctx context.Context, senderID uint64, senderNickname, senderAvatar string, receiverID uint64, articleID uint64, articleTitle string) error {
	// 1. 组装特有的 Content 结构体
	content := model.LikeArticleNotifyContent{
		ArticleID:    articleID,
		ArticleTitle: articleTitle,
	}

	// 2. 底层统一入口发送
	return s.send(ctx, model.NotifyTypeLikeArticle, senderID, senderNickname, senderAvatar, receiverID, content)
}
