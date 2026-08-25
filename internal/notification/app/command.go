package app

import "time"

// CreateArticleLikeNotificationCommand 表示文章点赞通知处理输入。
type CreateArticleLikeNotificationCommand struct {
	SenderID    uint64    // 点赞用户唯一标识
	ArticleID   uint64    // 被点赞文章唯一标识
	CreatedTime time.Time // 点赞消息创建时间
}
