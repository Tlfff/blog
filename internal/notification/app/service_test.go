package app

import (
	"blog/internal/notification/domain"
	"context"
	"testing"
	"time"
)

// fakeNotificationRepository 是通知应用测试使用的内存 Repository。
type fakeNotificationRepository struct {
	items []*domain.Notification // 已保存通知
}

// Insert 保存通知。
func (f *fakeNotificationRepository) Insert(_ context.Context, notification *domain.Notification) error {
	f.items = append(f.items, notification)
	return nil
}

// GetList 返回已保存通知。
func (f *fakeNotificationRepository) GetList(context.Context, uint64, int64, int64) ([]*domain.Notification, error) {
	return f.items, nil
}

// MarkAllAsRead 将全部通知标记为已读。
func (f *fakeNotificationRepository) MarkAllAsRead(context.Context, uint64) error {
	for _, item := range f.items {
		item.IsRead = true
	}
	return nil
}

// GetUnreadCount 返回未读通知数量。
func (f *fakeNotificationRepository) GetUnreadCount(context.Context, uint64) (int64, error) {
	var count int64
	for _, item := range f.items {
		if !item.IsRead {
			count++
		}
	}
	return count, nil
}

// fakeArticleQuery 返回固定文章快照。
type fakeArticleQuery struct{}

// FindByID 返回固定文章快照。
func (fakeArticleQuery) FindByID(context.Context, uint64) (*domain.ArticleInfo, error) {
	return &domain.ArticleInfo{ID: 3, AuthorID: 9, Title: "文章"}, nil
}

// fakeUserQuery 返回固定用户快照。
type fakeUserQuery struct{}

// FindUserByID 返回固定用户快照。
func (fakeUserQuery) FindUserByID(context.Context, uint64) (*domain.UserInfo, error) {
	return &domain.UserInfo{ID: 2, Nickname: "发送者", Avatar: "avatar"}, nil
}

// TestCreateArticleLikeNotification 验证当前基线文章点赞通知创建流程。
func TestCreateArticleLikeNotification(t *testing.T) {
	repository := &fakeNotificationRepository{}
	service := NewService(repository, fakeArticleQuery{}, fakeUserQuery{})
	createdTime := time.Unix(1_700_000_000, 0).UTC()
	if err := service.CreateLikeNotification(context.Background(), domain.NotificationEvent{
		NotifyType: domain.NotifyTypeLikeArticle.Int8(), SenderID: 2, TargetID: 3, CreatedTime: createdTime,
	}); err != nil {
		t.Fatalf("创建文章点赞通知失败: %v", err)
	}
	if len(repository.items) != 1 {
		t.Fatalf("通知数量不正确: %d", len(repository.items))
	}
	if repository.items[0].ReceiverID != 9 || repository.items[0].Sender.UserID != 2 {
		t.Fatalf("通知快照不正确: %+v", repository.items[0])
	}
}

// TestRejectsUnwiredNotificationType 验证未接线通知类型不会被新增处理。
func TestRejectsUnwiredNotificationType(t *testing.T) {
	service := NewService(&fakeNotificationRepository{}, fakeArticleQuery{}, fakeUserQuery{})
	if err := service.CreateLikeNotification(context.Background(), domain.NotificationEvent{NotifyType: domain.NotifyTypeLikeComment.Int8()}); err == nil {
		t.Fatal("未接线通知类型应被拒绝")
	}
}
