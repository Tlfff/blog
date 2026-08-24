package infra

import (
	"blog/internal/notification/domain"
	"testing"
	"time"
)

// TestNotificationMapperKeepsBaselineFields 验证 MongoDB Document 与领域对象转换不新增字段语义。
func TestNotificationMapperKeepsBaselineFields(t *testing.T) {
	createdTime := time.Unix(1_700_000_000, 0).UTC()
	notification := &domain.Notification{
		ReceiverID:  9,
		Sender:      domain.NotifySender{UserID: 2, Nickname: "发送者", Avatar: "avatar"},
		Type:        domain.NotifyTypeLikeArticle,
		Content:     domain.LikeArticleContent{ArticleID: 3, ArticleTitle: "文章"},
		CreatedTime: createdTime,
	}
	document := toModelNotification(notification)
	got := toDomainNotification(document)
	if got.ReceiverID != notification.ReceiverID || got.Sender.UserID != notification.Sender.UserID || got.Type != notification.Type {
		t.Fatalf("通知基础字段转换错误: %+v", got)
	}
	content, ok := got.Content.(domain.LikeArticleContent)
	if !ok || content.ArticleID != 3 || content.ArticleTitle != "文章" {
		t.Fatalf("通知内容转换错误: %#v", got.Content)
	}
	if !got.CreatedTime.Equal(createdTime) {
		t.Fatalf("通知时间转换错误: %v", got.CreatedTime)
	}
}
