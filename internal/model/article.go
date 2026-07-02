package model

import "time"

type Article struct {
	ID           uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`           // 文章唯一标识
	AuthorID     uint64    `gorm:"column:author_id" json:"author_id"`                      // 作者用户ID
	Title        string    `gorm:"column:title" json:"title"`                              // 文章标题
	Content      string    `gorm:"column:content" json:"content"`                          // 文章正文内容（支持Markdown）
	Tags         string    `gorm:"column:tags" json:"tags"`                                // 文章标签
	Status       int8      `gorm:"column:status" json:"status"`                            // 文章状态：0-已删除 1-草稿 2-已发表
	ViewCount    uint32    `gorm:"column:view_count" json:"view_count"`                    // 浏览量
	LikeCount    uint32    `gorm:"column:like_count" json:"like_count"`                    // 点赞数
	CommentCount uint32    `gorm:"column:comment_count" json:"comment_count"`              // 评论数
	CreatedTime  time.Time `gorm:"column:created_time;autoCreateTime" json:"created_time"` // 创建时间
	UpdatedTime  time.Time `gorm:"column:updated_time;autoUpdateTime" json:"updated_time"` // 最后更新时间

}
