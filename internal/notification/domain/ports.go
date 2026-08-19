package domain

import "context"

// NotificationRepository 是通知持久化 Port
type NotificationRepository interface {
	Insert(ctx context.Context, notification *Notification) error
	GetList(ctx context.Context, receiverID uint64, page, pageSize int64) ([]*Notification, error)
	MarkAllAsRead(ctx context.Context, receiverID uint64) error
	GetUnreadCount(ctx context.Context, receiverID uint64) (int64, error)
}

// ArticleInfo 是通知生成所需的文章查询模型
type ArticleInfo struct {
	ID       uint64 // 文章唯一标识
	AuthorID uint64 // 文章作者用户ID
	Title    string // 文章标题
}

// ArticleQuery 查询通知生成所需的文章信息
type ArticleQuery interface {
	FindByID(ctx context.Context, id uint64) (*ArticleInfo, error)
}

// UserInfo 是通知生成所需的用户查询模型
type UserInfo struct {
	ID       uint64 // 用户唯一标识
	Nickname string // 用户昵称
	Avatar   string // 用户头像URL
}

// UserInfoQuery 查询通知发送方公开信息
type UserInfoQuery interface {
	FindUserByID(ctx context.Context, id uint64) (*UserInfo, error)
}

// CommentInfo 是通知生成所需的评论查询模型
type CommentInfo struct {
	ID        uint64 // 评论唯一标识
	ArticleID uint64 // 所属文章ID
	UserID    uint64 // 评论作者用户ID
}

// CommentQuery 查询通知生成所需的评论信息
type CommentQuery interface {
	FindByID(ctx context.Context, id uint64) (*CommentInfo, error)
}
