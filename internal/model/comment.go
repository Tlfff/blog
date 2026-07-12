package model

import "time"

type Comment struct {
	ID            uint64    `gorm:"column:id;primaryKey;autoIncrement"` //评论ID
	ArticleID     uint64    `gorm:"column:article_id"`                  //文章ID
	UserID        uint64    `gorm:"column:user_id"`                     //评论用户ID
	ReplyToUserID uint64    `gorm:"column:reply_to_user_id"`            //被回复的用户ID
	Content       string    `gorm:"column:content"`                     //评论内容
	RootID        uint64    `gorm:"column:root_id"`                     //根评论ID
	IP            string    `gorm:"column:ip" `                         // 评论的IP地址
	LikeCount     int64     `gorm:"column:like_count"`                  //点赞数量
	CommentCount  int64     `gorm:"column:comment_count"`               //子评论数量（主评论特有）
	CreatedTime   time.Time `gorm:"column:created_time;autoCreateTime"` //评论创建时间
	UpdatedTime   time.Time `gorm:"column:updated_time;autoUpdateTime"` //评论更新时间
	Status        int8      `gorm:"column:status"`                      //评论状态：2-已删除 1-正常
}

func (Comment) TableName() string {
	return "comments"
}
