package kafka

import (
	"blog/internal/platform/kafka/message"
	"context"
	"encoding/json"
	"testing"
	"time"
)

// fakeConsumer 记录浏览历史消息。
type fakeConsumer struct {
	userID      uint64    // 浏览用户唯一标识
	articleID   uint64    // 被浏览文章唯一标识
	createdTime time.Time // 浏览时间
}

// CreateViewHistory 记录浏览历史消息。
func (f *fakeConsumer) CreateViewHistory(_ context.Context, userID, articleID uint64, timestamp time.Time) error {
	f.userID, f.articleID, f.createdTime = userID, articleID, timestamp
	return nil
}

// TestHandlerKeepsViewHistoryContract 验证浏览历史 Kafka Adapter 字段映射。
func TestHandlerKeepsViewHistoryContract(t *testing.T) {
	createdTime := time.Unix(1_700_000_000, 0).UTC()
	payload, err := json.Marshal(message.ViewHistoryMsg{UserID: 2, ArticleID: 3, CreatedTime: createdTime})
	if err != nil {
		t.Fatalf("序列化浏览历史消息失败: %v", err)
	}
	consumer := &fakeConsumer{}
	if err := NewHandler(consumer)(context.Background(), "2", payload); err != nil {
		t.Fatalf("处理浏览历史消息失败: %v", err)
	}
	if consumer.userID != 2 || consumer.articleID != 3 || !consumer.createdTime.Equal(createdTime) {
		t.Fatalf("浏览历史字段映射错误: %+v", consumer)
	}
}
