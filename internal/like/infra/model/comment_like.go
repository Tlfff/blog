package model

import "time"

// CommentLike 是评论点赞数据模型（GORM 映射 comment_likes 表）。
type CommentLike struct {
	ID          uint64    `gorm:"column:id;primaryKey;autoIncrement" ` // 唯一标识
	UserID      uint64    `gorm:"column:user_id" `                     // 用户ID
	CommentID   uint64    `gorm:"column:comment_id" `                  // 评论id
	Status      int8      `gorm:"column:status" `                      // 点赞状态：1-点赞；2-取消点赞
	CreatedTime time.Time `gorm:"column:created_time;autoCreateTime" ` // 创建时间
	UpdatedTime time.Time `gorm:"column:updated_time;autoUpdateTime" ` // 最后更新时间
}

// TableName 返回评论点赞模型对应的数据库表名。
func (CommentLike) TableName() string {
	return "comment_likes"
}

// 评论点赞状态取值
const (
	CommentLiked       = 1 // 点赞
	CommentCancelLiked = 2 // 取消点赞
)
