package community

import "time"

// 通知类型取值
const (
	NotifyTypeLikeArticle    int8 = 1 // 通知类型：点赞了文章
	NotifyTypeLikeComment    int8 = 2 // 通知类型：点赞了评论
	NotifyTypeCommentArticle int8 = 3 // 通知类型：评论了文章
	NotifyTypeReplyComment   int8 = 4 // 通知类型：回复了评论
)

// NotifySender 是通知发送方的公开信息（值对象）。
type NotifySender struct {
	UserID   uint64 // 发送方用户ID
	Nickname string // 发送方昵称
	Avatar   string // 发送方头像URL
}

// LikeArticleContent 是点赞文章类通知的内容载荷（值对象）。
type LikeArticleContent struct {
	ArticleID    uint64 // 被点赞的文章ID
	ArticleTitle string // 被点赞的文章标题
}

// Notification 是 Community 领域的通知聚合。
type Notification struct {
	ID          string       // 通知唯一标识（MongoDB ObjectID 字符串）
	ReceiverID  uint64       // 通知接收方用户ID
	Sender      NotifySender // 通知发送方公开信息
	Type        int8         // 通知类型：1-点赞文章 2-点赞评论 3-评论文章 4-回复评论
	IsRead      bool         // 是否已读
	Content     any          // 通知内容载荷，按 Type 取对应的内容结构
	CreatedTime time.Time    // 创建时间
}

// NotificationEvent 是点赞等行为触发的异步通知事件。
type NotificationEvent struct {
	NotifyType  int8      // 通知类型：1-点赞文章 2-点赞评论 3-评论文章 4-回复评论
	SenderID    uint64    // 触发行为的用户ID
	TargetID    uint64    // 行为目标ID，按 NotifyType 为文章ID或评论ID
	CreatedTime time.Time // 事件产生时间
}

// ViewHistoryEvent 是浏览文章触发的异步浏览历史事件。
type ViewHistoryEvent struct {
	ArticleID   uint64    // 被浏览的文章ID
	UserID      uint64    // 浏览者用户ID
	CreatedTime time.Time // 事件产生时间
}
