package community

import "context"

// CommentRepository 是评论持久化 Port。
type CommentRepository interface {
	CreateWithCounts(ctx context.Context, comment *Comment, incrementReply bool) error
	FindByID(ctx context.Context, id uint64) (*Comment, error)
	ListRootComments(ctx context.Context, articleID, lastID uint64, page, pageSize int, isDesc bool, authorID uint64) ([]*CommentWithUser, error)
	CountRootComments(ctx context.Context, articleID, authorID uint64) (int64, error)
	ListReplies(ctx context.Context, rootID, lastID uint64, page, pageSize int) ([]*CommentWithUser, error)
	CountReplies(ctx context.Context, rootID uint64) (int64, error)
	DeleteWithCounts(ctx context.Context, comment *Comment) error
}

// ArticleLikeRepository 是文章点赞持久化 Port。
type ArticleLikeRepository interface {
	SetLiked(ctx context.Context, userID, articleID uint64, liked bool) error
	IsLiked(ctx context.Context, userID, articleID uint64) (bool, error)
	GetLikedUserIDs(ctx context.Context, articleID uint64) ([]uint64, error)
}

// CommentLikeRepository 是评论点赞持久化 Port。
type CommentLikeRepository interface {
	SetLiked(ctx context.Context, userID, commentID uint64, liked bool) error
	IsLiked(ctx context.Context, userID, commentID uint64) (bool, error)
	GetLikedUserIDs(ctx context.Context, commentID uint64) ([]uint64, error)
}

// ViewHistoryRepository 是浏览历史持久化 Port。
type ViewHistoryRepository interface {
	Create(ctx context.Context, history *ViewHistory) error
	IncrementViewCount(ctx context.Context, articleID uint64) error
}

// NotificationRepository 是通知持久化 Port。
type NotificationRepository interface {
	Insert(ctx context.Context, notification *Notification) error
	GetList(ctx context.Context, receiverID uint64, page, pageSize int64) ([]*Notification, error)
	MarkAllAsRead(ctx context.Context, receiverID uint64) error
	GetUnreadCount(ctx context.Context, receiverID uint64) (int64, error)
}

// ArticleInfo 是 Community 查询文章所需的只读信息。
type ArticleInfo struct {
	ID           uint64
	AuthorID     uint64
	Title        string
	ViewCount    uint32
	LikeCount    uint32
	CommentCount uint32
}

// ArticleQuery 是 Community 的文章只读查询 Port。
type ArticleQuery interface {
	FindByID(ctx context.Context, id uint64) (*ArticleInfo, error)
	GetHotListByIDs(ctx context.Context, ids []uint64) ([]*ArticleInfo, error)
	GetTopHotArticles(ctx context.Context, limit int) ([]*ArticleInfo, error)
}

// UserInfo 是 Community 查询用户所需的公开信息。
type UserInfo struct {
	ID       uint64
	Nickname string
	Avatar   string
}

// UserInfoQuery 是 Community 的用户只读查询 Port。
type UserInfoQuery interface {
	FindUserByID(ctx context.Context, id uint64) (*UserInfo, error)
}

// LikeCache 是点赞状态缓存 Port，负责缓存命中与冷启动重建。
type LikeCache interface {
	IsLiked(ctx context.Context, target LikeTarget, targetID, userID uint64) (bool, error)
	Add(ctx context.Context, target LikeTarget, targetID, userID uint64) error
	Remove(ctx context.Context, target LikeTarget, targetID, userID uint64) error
}

// LikeCountStore 是评论点赞数读取 Port。
type LikeCountStore interface {
	GetCommentLikeCounts(ctx context.Context, commentIDs []uint64) (map[uint64]uint64, error)
}

// HotRankStore 是热榜 Redis Port。
type HotRankStore interface {
	GetTop(ctx context.Context, limit int) ([]HotRankItem, error)
	Rebuild(ctx context.Context, entries []HotRankItem) error
}

// EventPublisher 是 Community 异步事件发布 Port。
type EventPublisher interface {
	SendLikeNotification(ctx context.Context, event NotificationEvent) error
	SendViewHistory(ctx context.Context, event ViewHistoryEvent) error
}
