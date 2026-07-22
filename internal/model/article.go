package model

import "time"

type Article struct {
	ID           uint64    `gorm:"column:id;primaryKey;autoIncrement" ` // 文章唯一标识
	AuthorID     uint64    `gorm:"column:author_id" `                   // 作者用户ID
	Title        string    `gorm:"column:title" `                       // 文章标题
	Content      string    `gorm:"column:content" `                     // 文章正文内容（支持Markdown）
	Tags         string    `gorm:"column:tags" `                        // 文章标签
	Status       int8      `gorm:"column:status" `                      // 文章状态：1-已删除 2-草稿 3-已发表
	ViewCount    uint32    `gorm:"column:view_count" `                  // 浏览量
	LikeCount    uint32    `gorm:"column:like_count"`                   // 点赞数
	CommentCount uint32    `gorm:"column:comment_count" `               // 评论数
	CreatedTime  time.Time `gorm:"column:created_time;autoCreateTime" ` // 创建时间
	UpdatedTime  time.Time `gorm:"column:updated_time;autoUpdateTime" ` // 最后更新时间

}
