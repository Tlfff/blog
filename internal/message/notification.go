package message

import "time"

// NotificationMsg 是通知 Kafka 消息体（DTO）。
type NotificationMsg struct {
	NotifyType  int8      `json:"notify_type"`  // 通知类型，和model.Type一致
	SenderID    uint64    `json:"sender_id"`    // 操作人ID
	TargetID    uint64    `json:"target_id"`    // 目标ID（文章ID、评论ID等）
	CreatedTime time.Time `json:"created_time"` // 消息创建时间
}
