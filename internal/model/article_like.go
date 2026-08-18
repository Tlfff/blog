package model

import "time"

// ArticleLike 是文章点赞数据模型（GORM 映射 article_likes 表）。
type ArticleLike struct {
	ID          uint64    `gorm:"column:id;primaryKey;autoIncrement" ` // 唯一标识
	UserID      uint64    `gorm:"column:user_id" `                     // 用户ID
	ArticleID   uint64    `gorm:"column:article_id" `                  // 文章id
	Status      int8      `gorm:"column:status" `                      // 点赞状态：1-点赞；2-取消点赞
	CreatedTime time.Time `gorm:"column:created_time;autoCreateTime" ` // 创建时间
	UpdatedTime time.Time `gorm:"column:updated_time;autoUpdateTime" ` // 最后更新时间
}

// 指定该模型对应的数据库表名
func (ArticleLike) TableName() string {
	return "article_likes"
}

// 文章点赞状态取值
const (
	ArticleLiked       = 1 // 点赞
	ArticleCancelLiked = 2 // 取消点赞
)
