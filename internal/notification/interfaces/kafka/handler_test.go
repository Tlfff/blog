package kafka

import (
	notificationdomain "blog/internal/notification/domain"
	"blog/internal/platform/kafka/message"
	"context"
	"encoding/json"
	"testing"
	"time"
)

// fakeConsumer 记录 Kafka Handler 转换后的通知事件。
type fakeConsumer struct {
	event notificationdomain.NotificationEvent // 最近一次收到的通知事件
}

// CreateLikeNotification 记录收到的通知事件。
func (f *fakeConsumer) CreateLikeNotification(_ context.Context, event notificationdomain.NotificationEvent) error {
	f.event = event
	return nil
}

// TestHandlerKeepsBaselineMessageContract 验证 Handler 只解析基线已有字段。
func TestHandlerKeepsBaselineMessageContract(t *testing.T) {
	createdTime := time.Unix(1_700_000_000, 0).UTC()
	payload, err := json.Marshal(message.NotificationMsg{
		NotifyType:  1,
		SenderID:    12,
		TargetID:    34,
		CreatedTime: createdTime,
	})
	if err != nil {
		t.Fatalf("序列化通知消息失败: %v", err)
	}

	consumer := &fakeConsumer{}
	if err := NewHandler(consumer)(context.Background(), "", payload); err != nil {
		t.Fatalf("处理通知消息失败: %v", err)
	}
	if consumer.event.NotifyType != 1 || consumer.event.SenderID != 12 || consumer.event.TargetID != 34 {
		t.Fatalf("通知事件字段转换错误: %+v", consumer.event)
	}
	if !consumer.event.CreatedTime.Equal(createdTime) {
		t.Fatalf("通知创建时间错误: got=%v want=%v", consumer.event.CreatedTime, createdTime)
	}
}
