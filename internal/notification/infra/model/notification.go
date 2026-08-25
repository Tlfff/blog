package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// NotifySender 表示 MongoDB 中保存的通知发送方快照。
type NotifySender struct {
	UserID   uint64 `bson:"user_id"`  // 用户唯一标识
	Nickname string `bson:"nickname"` // 用户昵称
	Avatar   string `bson:"avatar"`   // 用户头像地址
}

// LikeArticleNotifyContent 表示 MongoDB 中保存的文章点赞内容。
type LikeArticleNotifyContent struct {
	ArticleID    uint64 `bson:"article_id"`    // 文章唯一标识
	ArticleTitle string `bson:"article_title"` // 文章标题
}

// Notification 表示 notifications 集合的文档模型。
type Notification struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"` // MongoDB 文档主键
	ReceiverID  uint64             `bson:"receiver_id"`   // 通知接收方用户唯一标识
	Sender      NotifySender       `bson:"sender"`        // 通知发送方快照
	Type        int8               `bson:"type"`          // 通知类型：1-点赞文章；2-点赞评论；3-评论文章；4-回复评论
	IsRead      bool               `bson:"is_read"`       // 是否已读
	Content     any                `bson:"content"`       // 通知内容快照
	CreatedTime time.Time          `bson:"created_time"`  // 通知创建时间
}
