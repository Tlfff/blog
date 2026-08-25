package domain

import "time"

// ViewHistory 是 Article 领域的浏览历史实体。
type ViewHistory struct {
	ID          uint64    // 浏览记录唯一标识
	UserID      uint64    // 浏览者用户ID
	ArticleID   uint64    // 被浏览的文章ID
	CreatedTime time.Time // 创建时间
	UpdatedTime time.Time // 最后更新时间
}
