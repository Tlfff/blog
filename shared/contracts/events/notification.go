package events

import "time"

// NotificationEvent 是点赞等业务事件的通知载荷。
type NotificationEvent struct {
	NotifyType  int8      `json:"notify_type"`  // 通知类型：1-点赞文章 2-点赞评论 3-评论文章 4-回复评论
	SenderID    uint64    `json:"sender_id"`    // 操作人（发送方）用户ID
	TargetID    uint64    `json:"target_id"`    // 目标ID（文章ID、评论ID等）
	CreatedTime time.Time `json:"created_time"` // 事件创建时间
}

// ViewHistoryEvent 是浏览历史事件载荷。
type ViewHistoryEvent struct {
	ArticleID   uint64    `json:"article_id"`   // 被浏览的文章ID
	UserID      uint64    `json:"user_id"`      // 浏览者用户ID
	CreatedTime time.Time `json:"created_time"` // 事件创建时间
}
