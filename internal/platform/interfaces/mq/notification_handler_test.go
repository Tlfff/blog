package mq

import (
	domain "blog/internal/notification/domain"
	"context"
	"encoding/json"
	"testing"

	message "blog/shared/contracts/events/messages"
)

type notificationConsumerStub struct{}

func (notificationConsumerStub) CreateLikeNotification(context.Context, domain.NotificationEvent) error {
	return nil
}

func TestNotificationHandlerRejectsUnsupportedVersion(t *testing.T) {
	payload, err := json.Marshal(message.NotificationMsg{EventID: "evt-1", Version: 2})
	if err != nil {
		t.Fatal(err)
	}
	if err := NewNotificationHandler(notificationConsumerStub{})(context.Background(), "", payload); err == nil {
		t.Fatal("不支持的通知事件版本应被拒绝")
	}
}

func TestNotificationHandlerAcceptsVersionOne(t *testing.T) {
	payload, err := json.Marshal(message.NotificationMsg{EventID: "evt-1", Version: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := NewNotificationHandler(notificationConsumerStub{})(context.Background(), "", payload); err != nil {
		t.Fatalf("版本1通知事件不应失败: %v", err)
	}
}
