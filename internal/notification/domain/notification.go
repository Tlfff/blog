package domain

import (
	"errors"
	"time"
)

// NotificationType 表示通知类型值对象。
type NotificationType int8

const (
	NotifyTypeLikeArticle    NotificationType = 1 // 通知类型：点赞文章
	NotifyTypeLikeComment    NotificationType = 2 // 兼容存量数据：点赞评论
	NotifyTypeCommentArticle NotificationType = 3 // 兼容存量数据：评论文章
	NotifyTypeReplyComment   NotificationType = 4 // 兼容存量数据：回复评论
)

var (
	// ErrInvalidNotificationType 表示通知类型值不在兼容集合中。
	ErrInvalidNotificationType = errors.New("通知类型非法")
	// ErrUnsupportedNotificationType 表示通知类型存在但当前未接入创建流程。
	ErrUnsupportedNotificationType = errors.New("不支持的通知类型")
)

// NewNotificationType 创建兼容存量数据的通知类型。
func NewNotificationType(value int8) (NotificationType, error) {
	notifyType := NotificationType(value)
	switch notifyType {
	case NotifyTypeLikeArticle, NotifyTypeLikeComment, NotifyTypeCommentArticle, NotifyTypeReplyComment:
		return notifyType, nil
	default:
		return 0, ErrInvalidNotificationType
	}
}

// Int8 返回 MongoDB 和协议兼容所需的类型值。
func (t NotificationType) Int8() int8 {
	return int8(t)
}

// EnsureCreatable 校验当前通知类型是否已接入创建流程。
func (t NotificationType) EnsureCreatable() error {
	if t != NotifyTypeLikeArticle {
		return ErrUnsupportedNotificationType
	}
	return nil
}

// NotifySender 表示通知发送方的公开快照。
type NotifySender struct {
	UserID   uint64 // 发送方用户唯一标识
	Nickname string // 发送方昵称
	Avatar   string // 发送方头像地址
}

// LikeArticleContent 表示文章点赞通知的内容快照。
type LikeArticleContent struct {
	ArticleID    uint64 // 被点赞文章唯一标识
	ArticleTitle string // 被点赞文章标题
}

// Notification 是通知聚合。
type Notification struct {
	ID          string           // MongoDB ObjectID 的十六进制字符串
	ReceiverID  uint64           // 通知接收方用户唯一标识
	Sender      NotifySender     // 通知发送方公开快照
	Type        NotificationType // 通知类型
	IsRead      bool             // 是否已读
	Content     any              // 按通知类型保存的内容快照
	CreatedTime time.Time        // 通知创建时间
}

// NotificationEvent 表示当前 Kafka 基线中的通知消息。
type NotificationEvent struct {
	NotifyType  int8      // 通知类型
	SenderID    uint64    // 操作用户唯一标识
	TargetID    uint64    // 目标文章或评论唯一标识
	CreatedTime time.Time // 消息创建时间
}

// NewArticleLikeNotification 创建当前已接线的文章点赞通知。
func NewArticleLikeNotification(notifyType NotificationType, sender UserInfo, article ArticleInfo, createdTime time.Time) (*Notification, bool, error) {
	// 1. 只允许当前已接线的文章点赞通知
	if err := notifyType.EnsureCreatable(); err != nil {
		return nil, false, err
	}

	// 2. 自己点赞自己的文章时不创建通知
	if sender.ID == article.AuthorID {
		return nil, false, nil
	}

	// 3. 组装未读通知快照
	return &Notification{
		ReceiverID: article.AuthorID,
		Sender: NotifySender{
			UserID:   sender.ID,
			Nickname: sender.Nickname,
			Avatar:   sender.Avatar,
		},
		Type:   notifyType,
		IsRead: false,
		Content: LikeArticleContent{
			ArticleID:    article.ID,
			ArticleTitle: article.Title,
		},
		CreatedTime: createdTime,
	}, true, nil
}
