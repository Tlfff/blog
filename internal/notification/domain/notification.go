package domain

import "time"

const (
	NotifyTypeLikeArticle    int8 = 1 // 通知类型：点赞文章
	NotifyTypeLikeComment    int8 = 2 // 兼容存量数据：点赞评论
	NotifyTypeCommentArticle int8 = 3 // 兼容存量数据：评论文章
	NotifyTypeReplyComment   int8 = 4 // 兼容存量数据：回复评论
)

// NotifySender 表示通知发送方的公开快照。
type NotifySender struct {
	UserID   uint64 // 发送方用户唯一标识
	Nickname string // 发送方昵称
	Avatar   string // 发送方头像地址
}

// LikeArticleContent 表示文章点赞通知的内容快照。
type LikeArticleContent struct {
	ArticleID    uint64 // 被点赞文章唯一标识
	ArticleTitle string // 被点赞文章标题
}

// Notification 表示通知聚合。
type Notification struct {
	ID          string       // MongoDB ObjectID 的十六进制字符串
	ReceiverID  uint64       // 通知接收方用户唯一标识
	Sender      NotifySender // 通知发送方公开快照
	Type        int8         // 通知类型：1-点赞文章；2-点赞评论；3-评论文章；4-回复评论
	IsRead      bool         // 是否已读
	Content     any          // 按通知类型保存的内容快照
	CreatedTime time.Time    // 通知创建时间
}

// NotificationEvent 表示当前 Kafka 基线中的通知消息。
type NotificationEvent struct {
	NotifyType  int8      // 通知类型
	SenderID    uint64    // 操作用户唯一标识
	TargetID    uint64    // 目标文章或评论唯一标识
	CreatedTime time.Time // 消息创建时间
}
