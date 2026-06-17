package model

import "time"

type Article struct {
	ID         int64     `json:"id"`          // 文章唯一标识
	AuthorID   int64     `json:"author_id"`   // 作者用户ID
	Title      string    `json:"title"`       // 文章标题
	Content    string    `json:"content"`     // 文章正文内容（支持Markdown）
	Tags       []string  `json:"tags"`        // 文章标签
	Status     int8      `json:"status"`      // 文章状态：0-已删除 1-草稿 2-已发表
	AddTime    time.Time `json:"add_time"`    // 创建时间
	UpdateTime time.Time `json:"update_time"` // 最后更新时间
}
