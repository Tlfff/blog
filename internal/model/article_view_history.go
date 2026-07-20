package model

import "time"

type ArticleViewHistory struct {
	ID          uint64    `gorm:"primaryKey;column:id" `               // 主键ID
	UserID      uint64    `gorm:"column:user_id" `                     // 用户ID
	ArticleID   uint64    `gorm:"column:article_id" `                  // 文章ID
	CreatedTime time.Time `gorm:"column:created_time;autoCreateTime" ` // 创建时间
	UpdatedTime time.Time `gorm:"column:updated_time;autoUpdateTime" ` // 更新时间，公司规范保留字段
}
