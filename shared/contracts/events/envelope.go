// Package events 定义跨服务 Kafka 事件的统一信封。
package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	TypeNotification = "notification"
	TypeViewHistory  = "view_history"
	VersionV1        = "v1"
)

// Envelope 是所有 Kafka 消息的统一信封，包含幂等与版本信息。
type Envelope struct {
	EventID    string          `json:"event_id"`
	EventType  string          `json:"event_type"`
	Version    string          `json:"version"`
	OccurredAt time.Time       `json:"occurred_at"`
	Payload    json.RawMessage `json:"payload"`
}

// NewEnvelope 构造带幂等 ID 的事件信封。
func NewEnvelope(eventType, version string, payload any) (*Envelope, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &Envelope{
		EventID:    uuid.New().String(),
		EventType:  eventType,
		Version:    version,
		OccurredAt: time.Now().UTC(),
		Payload:    data,
	}, nil
}

// Marshal 序列化为 JSON 字节。
func (e *Envelope) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

// DecodeEnvelope 从 JSON 字节解析信封。
func DecodeEnvelope(data []byte) (*Envelope, error) {
	var envelope Envelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, err
	}
	return &envelope, nil
}

// UnmarshalPayload 把 payload 解码到目标结构。
func (e *Envelope) UnmarshalPayload(v any) error {
	return json.Unmarshal(e.Payload, v)
}
