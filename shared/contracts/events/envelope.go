// Package events 定义跨服务 Kafka 事件的统一信封。
package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// 事件类型与信封版本常量
const (
	TypeNotification = "notification" // 事件类型：通知类事件
	TypeViewHistory  = "view_history" // 事件类型：浏览历史事件
	VersionV1        = "v1"           // 事件信封版本号
)

// Envelope 是所有 Kafka 消息的统一信封，包含幂等与版本信息。
type Envelope struct {
	EventID    string          `json:"event_id"`    // 事件唯一ID，用于消费端幂等去重
	EventType  string          `json:"event_type"`  // 事件类型：notification-通知 view_history-浏览历史
	Version    string          `json:"version"`     // 信封版本号，便于后续兼容升级
	OccurredAt time.Time       `json:"occurred_at"` // 事件发生时间（UTC）
	Payload    json.RawMessage `json:"payload"`     // 业务载荷原始JSON，由各事件结构自行解码
}

// 构造带幂等ID的事件信封，自动生成 EventID 并记录发生时间
func NewEnvelope(eventType, version string, payload any) (*Envelope, error) {
	// 1. 序列化业务载荷为 JSON
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	// 2. 填充幂等ID、事件类型、版本与发生时间
	return &Envelope{
		EventID:    uuid.New().String(),
		EventType:  eventType,
		Version:    version,
		OccurredAt: time.Now().UTC(),
		Payload:    data,
	}, nil
}

// 将信封序列化为 JSON 字节
func (e *Envelope) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

// 从 JSON 字节解析出事件信封
func DecodeEnvelope(data []byte) (*Envelope, error) {
	var envelope Envelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, err
	}
	return &envelope, nil
}

// 将信封中的 payload 解码到目标结构
func (e *Envelope) UnmarshalPayload(v any) error {
	return json.Unmarshal(e.Payload, v)
}
