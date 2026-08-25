package message

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"
	"time"
)

// TestNotificationMessageJSONContract 验证通知消息字段保持基线兼容。
func TestNotificationMessageJSONContract(t *testing.T) {
	payload, err := json.Marshal(NotificationMsg{NotifyType: 1, SenderID: 2, TargetID: 3, CreatedTime: time.Unix(0, 0).UTC()})
	if err != nil {
		t.Fatalf("序列化通知消息失败: %v", err)
	}
	assertJSONKeys(t, payload, []string{"created_time", "notify_type", "sender_id", "target_id"})
}

// TestViewHistoryMessageJSONContract 验证浏览历史消息字段保持基线兼容。
func TestViewHistoryMessageJSONContract(t *testing.T) {
	payload, err := json.Marshal(ViewHistoryMsg{ArticleID: 3, UserID: 2, CreatedTime: time.Unix(0, 0).UTC()})
	if err != nil {
		t.Fatalf("序列化浏览历史消息失败: %v", err)
	}
	assertJSONKeys(t, payload, []string{"article_id", "created_time", "user_id"})
}

// assertJSONKeys 断言 JSON 对象字段集合。
func assertJSONKeys(t *testing.T, payload []byte, want []string) {
	t.Helper()
	var object map[string]any
	if err := json.Unmarshal(payload, &object); err != nil {
		t.Fatalf("解析 JSON 失败: %v", err)
	}
	got := make([]string, 0, len(object))
	for key := range object {
		got = append(got, key)
	}
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("JSON 字段发生变化: got=%v want=%v", got, want)
	}
}
