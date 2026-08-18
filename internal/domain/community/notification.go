package community

import "time"

const (
	NotifyTypeLikeArticle    int8 = 1
	NotifyTypeLikeComment    int8 = 2
	NotifyTypeCommentArticle int8 = 3
	NotifyTypeReplyComment   int8 = 4
)

// NotifySender 是通知发送方的公开信息。
type NotifySender struct {
	UserID   uint64
	Nickname string
	Avatar   string
}

// LikeArticleContent 是点赞文章通知内容。
type LikeArticleContent struct {
	ArticleID    uint64
	ArticleTitle string
}

// Notification 是通知聚合。
type Notification struct {
	ID          string
	ReceiverID  uint64
	Sender      NotifySender
	Type        int8
	IsRead      bool
	Content     any
	CreatedTime time.Time
}

// NotificationEvent 是点赞等异步通知事件。
type NotificationEvent struct {
	NotifyType  int8
	SenderID    uint64
	TargetID    uint64
	CreatedTime time.Time
}

// ViewHistoryEvent 是浏览历史异步事件。
type ViewHistoryEvent struct {
	ArticleID   uint64
	UserID      uint64
	CreatedTime time.Time
}
