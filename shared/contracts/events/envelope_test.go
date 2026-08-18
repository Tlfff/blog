package events

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEnvelopeRoundTrip(t *testing.T) {
	payload := NotificationEvent{NotifyType: 1, SenderID: 2, TargetID: 3, CreatedTime: time.Now()}
	envelope, err := NewEnvelope(TypeNotification, VersionV1, payload)
	if err != nil {
		t.Fatalf("创建信封失败: %v", err)
	}
	if envelope.EventID == "" || envelope.EventType != TypeNotification || envelope.Version != VersionV1 {
		t.Fatalf("信封字段不完整: %+v", envelope)
	}

	data, err := envelope.Marshal()
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}
	var fields map[string]any
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}
	for _, key := range []string{"event_id", "event_type", "version", "occurred_at", "payload"} {
		if _, ok := fields[key]; !ok {
			t.Errorf("信封缺少字段 %s", key)
		}
	}

	decoded, err := DecodeEnvelope(data)
	if err != nil {
		t.Fatalf("解码信封失败: %v", err)
	}
	var decodedPayload NotificationEvent
	if err := decoded.UnmarshalPayload(&decodedPayload); err != nil {
		t.Fatalf("解码 payload 失败: %v", err)
	}
	if decodedPayload.SenderID != 2 || decodedPayload.TargetID != 3 {
		t.Fatalf("payload 不一致: %+v", decodedPayload)
	}
}
