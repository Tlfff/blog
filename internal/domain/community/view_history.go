package community

import "time"

// ViewHistory 是浏览历史流水。
type ViewHistory struct {
	ID          uint64
	UserID      uint64
	ArticleID   uint64
	CreatedTime time.Time
	UpdatedTime time.Time
}
