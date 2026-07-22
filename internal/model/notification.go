// internal/model/notification.go
package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	NotifyTypeLikeArticle    = 1 // 点赞文章通知
	NotifyTypeLikeComment    = 2 // 点赞评论通知
	NotifyTypeCommentArticle = 3 // 评论文章通知
	NotifyTypeReplyComment   = 4 // 回复评论通知
)

// 触发通知的发送方简要信息
type NotifySender struct {
	UserID   uint64 `bson:"user_id" `  // 用户ID
	Nickname string `bson:"nickname" ` // 用户昵称
	Avatar   string `bson:"avatar" `   // 用户头像
}

// 点赞文章特有的内容结构体
type LikeArticleNotifyContent struct {
	ArticleID    uint64 `bson:"article_id" `    // 文章ID
	ArticleTitle string `bson:"article_title" ` // 文章标题
}

// 消息主文档
type Notification struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" `
	ReceiverID  uint64             `bson:"receiver_id" `  // 消息接收者（文章作者）
	Sender      NotifySender       `bson:"sender" `       // 点赞的人
	Type        int8               `bson:"type" `         // 类型：1-点赞文章，2-点赞评论，3-评论文章，4-回复评论
	IsRead      bool               `bson:"is_read" `      // 是否已读
	Content     any                `bson:"content" `      // 动态内容
	CreatedTime time.Time          `bson:"created_time" ` // 创建时间
}
