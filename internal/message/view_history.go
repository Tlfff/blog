package message

import "time"

type ViewHistoryMsg struct {
	ArticleID   uint64    `json:"article_id"`   // 浏览的文章ID
	UserID      uint64    `json:"user_id"`      // 浏览用户ID
	CreatedTime time.Time `json:"created_time"` // 创建消息时间
}
