package domain

import (
	"errors"
	"testing"
	"time"
)

// TestNotificationType 验证通知类型集合和当前可创建类型。
func TestNotificationType(t *testing.T) {
	notifyType, err := NewNotificationType(NotifyTypeLikeArticle.Int8())
	if err != nil || notifyType != NotifyTypeLikeArticle {
		t.Fatalf("文章点赞通知类型创建失败: %v", err)
	}
	if _, err := NewNotificationType(99); !errors.Is(err, ErrInvalidNotificationType) {
		t.Fatalf("非法通知类型错误不正确: %v", err)
	}
	if err := NotifyTypeLikeComment.EnsureCreatable(); !errors.Is(err, ErrUnsupportedNotificationType) {
		t.Fatalf("未接线通知类型错误不正确: %v", err)
	}
}

// TestNewArticleLikeNotification 验证文章点赞通知工厂和自通知规则。
func TestNewArticleLikeNotification(t *testing.T) {
	createdTime := time.Unix(1_700_000_000, 0).UTC()
	notification, shouldCreate, err := NewArticleLikeNotification(
		NotifyTypeLikeArticle,
		UserInfo{ID: 2, Nickname: "发送者", Avatar: "avatar"},
		ArticleInfo{ID: 3, AuthorID: 9, Title: "文章"},
		createdTime,
	)
	if err != nil || !shouldCreate {
		t.Fatalf("创建文章点赞通知失败: %v", err)
	}
	if notification.ReceiverID != 9 || notification.Sender.UserID != 2 || notification.IsRead {
		t.Fatalf("通知快照错误: %+v", notification)
	}
	if !notification.CreatedTime.Equal(createdTime) {
		t.Fatalf("通知时间错误: %v", notification.CreatedTime)
	}

	notification, shouldCreate, err = NewArticleLikeNotification(
		NotifyTypeLikeArticle,
		UserInfo{ID: 9},
		ArticleInfo{ID: 3, AuthorID: 9},
		createdTime,
	)
	if err != nil || shouldCreate || notification != nil {
		t.Fatalf("自通知应被跳过: notification=%+v create=%v err=%v", notification, shouldCreate, err)
	}
}
