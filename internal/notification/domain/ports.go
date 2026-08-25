package domain

import "context"

// NotificationRepository 定义通知持久化能力。
type NotificationRepository interface {
	// Insert 写入一条通知。
	Insert(ctx context.Context, notification *Notification) error
	// GetList 按接收者分页查询通知。
	GetList(ctx context.Context, receiverID uint64, page, pageSize int64) ([]*Notification, error)
	// MarkAllAsRead 将接收者的全部未读通知标记为已读。
	MarkAllAsRead(ctx context.Context, receiverID uint64) error
	// GetUnreadCount 查询接收者的未读通知数量。
	GetUnreadCount(ctx context.Context, receiverID uint64) (int64, error)
}

// ArticleInfo 表示通知生成所需的最小文章快照。
type ArticleInfo struct {
	ID       uint64 // 文章唯一标识
	AuthorID uint64 // 文章作者用户唯一标识
	Title    string // 文章标题
}

// ArticleQuery 查询通知生成所需的文章信息。
type ArticleQuery interface {
	// FindByID 根据唯一标识查询文章快照。
	FindByID(ctx context.Context, id uint64) (*ArticleInfo, error)
}

// UserInfo 表示通知生成所需的最小用户快照。
type UserInfo struct {
	ID       uint64 // 用户唯一标识
	Nickname string // 用户昵称
	Avatar   string // 用户头像地址
}

// UserInfoQuery 查询通知发送方公开信息。
type UserInfoQuery interface {
	// FindUserByID 根据唯一标识查询用户公开快照。
	FindUserByID(ctx context.Context, id uint64) (*UserInfo, error)
}
