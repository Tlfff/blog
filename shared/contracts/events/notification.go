package events

import "time"

// NotificationEvent 是点赞等业务事件的通知载荷。
type NotificationEvent struct {
	NotifyType  int8      `json:"notify_type"`
	SenderID    uint64    `json:"sender_id"`
	TargetID    uint64    `json:"target_id"`
	CreatedTime time.Time `json:"created_time"`
}

// ViewHistoryEvent 是浏览历史事件载荷。
type ViewHistoryEvent struct {
	ArticleID   uint64    `json:"article_id"`
	UserID      uint64    `json:"user_id"`
	CreatedTime time.Time `json:"created_time"`
}
