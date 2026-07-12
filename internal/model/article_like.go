package model

import "time"

type ArticleLike struct {
	ID          uint64    `gorm:"column:id;primaryKey;autoIncrement" ` // 唯一标识
	UserID      uint64    `gorm:"column:author_id" `                   // 用户ID
	ArticleID   uint64    `gorm:"column:article_id" `                  // 文章id
	Status      int8      `gorm:"column:status" `                      // 点赞状态：1-点赞；2-取消点赞
	CreatedTime time.Time `gorm:"column:created_time;autoCreateTime" ` // 创建时间
	UpdatedTime time.Time `gorm:"column:updated_time;autoUpdateTime" ` // 最后更新时间
}

func (ArticleLike) TableName() string {
	return "article_likes"
}
