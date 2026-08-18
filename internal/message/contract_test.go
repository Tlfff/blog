package message

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNotificationMsgJSONContract(t *testing.T) {
	msg := NotificationMsg{
		NotifyType:  1,
		SenderID:    2,
		TargetID:    3,
		CreatedTime: time.Unix(1700000000, 0),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}

	var fields map[string]any
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}
	for _, key := range []string{"notify_type", "sender_id", "target_id", "created_time"} {
		if _, ok := fields[key]; !ok {
			t.Errorf("NotificationMsg 缺少字段 %s", key)
		}
	}
}

func TestViewHistoryMsgJSONContract(t *testing.T) {
	msg := ViewHistoryMsg{
		ArticleID:   1,
		UserID:      2,
		CreatedTime: time.Unix(1700000000, 0),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}

	var fields map[string]any
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}
	for _, key := range []string{"article_id", "user_id", "created_time"} {
		if _, ok := fields[key]; !ok {
			t.Errorf("ViewHistoryMsg 缺少字段 %s", key)
		}
	}
}
