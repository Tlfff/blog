package community

import "context"

// CommentRepository 是评论持久化 Port。
type CommentRepository interface {
	// 创建评论，并按需同步累加主楼评论的回复数
	CreateWithCounts(ctx context.Context, comment *Comment, incrementReply bool) error
	// 按评论ID查询单条评论
	FindByID(ctx context.Context, id uint64) (*Comment, error)
	// 分页查询文章下的主楼评论，附带作者公开信息
	ListRootComments(ctx context.Context, articleID, lastID uint64, page, pageSize int, isDesc bool, authorID uint64) ([]*CommentWithUser, error)
	// 统计文章下的主楼评论总数
	CountRootComments(ctx context.Context, articleID, authorID uint64) (int64, error)
	// 分页查询指定主楼下的回复列表，附带作者与被回复者公开信息
	ListReplies(ctx context.Context, rootID, lastID uint64, page, pageSize int) ([]*CommentWithUser, error)
	// 统计指定主楼下的回复总数
	CountReplies(ctx context.Context, rootID uint64) (int64, error)
	// 删除评论，并同步扣减文章评论数与主楼回复数
	DeleteWithCounts(ctx context.Context, comment *Comment) error
}

// ArticleLikeRepository 是文章点赞持久化 Port。
type ArticleLikeRepository interface {
	// 设置用户对文章的点赞状态（点赞或取消点赞）
	SetLiked(ctx context.Context, userID, articleID uint64, liked bool) error
	// 查询用户是否已点赞该文章
	IsLiked(ctx context.Context, userID, articleID uint64) (bool, error)
	// 查询点赞过该文章的全部用户ID，用于缓存冷启动重建
	GetLikedUserIDs(ctx context.Context, articleID uint64) ([]uint64, error)
}

// CommentLikeRepository 是评论点赞持久化 Port。
type CommentLikeRepository interface {
	// 设置用户对评论的点赞状态（点赞或取消点赞）
	SetLiked(ctx context.Context, userID, commentID uint64, liked bool) error
	// 查询用户是否已点赞该评论
	IsLiked(ctx context.Context, userID, commentID uint64) (bool, error)
	// 查询点赞过该评论的全部用户ID，用于缓存冷启动重建
	GetLikedUserIDs(ctx context.Context, commentID uint64) ([]uint64, error)
}

// ViewHistoryRepository 是浏览历史持久化 Port。
type ViewHistoryRepository interface {
	// 写入一条浏览历史流水
	Create(ctx context.Context, history *ViewHistory) error
	// 累加文章浏览量
	IncrementViewCount(ctx context.Context, articleID uint64) error
}

// NotificationRepository 是通知持久化 Port。
type NotificationRepository interface {
	// 插入一条通知
	Insert(ctx context.Context, notification *Notification) error
	// 分页查询用户的通知列表
	GetList(ctx context.Context, receiverID uint64, page, pageSize int64) ([]*Notification, error)
	// 将用户的全部未读通知标记为已读
	MarkAllAsRead(ctx context.Context, receiverID uint64) error
	// 统计用户的未读通知数
	GetUnreadCount(ctx context.Context, receiverID uint64) (int64, error)
}

// ArticleInfo 是 Community 查询文章所需的只读信息（查询模型）。
type ArticleInfo struct {
	ID           uint64 // 文章ID
	AuthorID     uint64 // 作者用户ID
	Title        string // 文章标题
	ViewCount    uint32 // 浏览量
	LikeCount    uint32 // 点赞数
	CommentCount uint32 // 评论数
}

// ArticleQuery 是 Community 的文章只读查询 Port，由 Content 侧提供。
type ArticleQuery interface {
	// 按文章ID查询文章只读信息
	FindByID(ctx context.Context, id uint64) (*ArticleInfo, error)
	// 按文章ID批量查询文章只读信息，用于补全热榜标题与统计
	GetHotListByIDs(ctx context.Context, ids []uint64) ([]*ArticleInfo, error)
	// 查询热度最高的文章列表，用于热榜冷启动重建
	GetTopHotArticles(ctx context.Context, limit int) ([]*ArticleInfo, error)
}

// UserInfo 是 Community 查询用户所需的公开信息（查询模型）。
type UserInfo struct {
	ID       uint64 // 用户ID
	Nickname string // 用户昵称
	Avatar   string // 用户头像URL
}

// UserInfoQuery 是 Community 的用户只读查询 Port，由 Identity 侧提供。
type UserInfoQuery interface {
	// 按用户ID查询用户公开信息
	FindUserByID(ctx context.Context, id uint64) (*UserInfo, error)
}

// LikeCache 是点赞状态缓存 Port，负责缓存命中与冷启动重建。
type LikeCache interface {
	// 查询缓存中用户是否已点赞指定目标
	IsLiked(ctx context.Context, target LikeTarget, targetID, userID uint64) (bool, error)
	// 将用户加入指定目标的点赞集合
	Add(ctx context.Context, target LikeTarget, targetID, userID uint64) error
	// 将用户从指定目标的点赞集合中移除
	Remove(ctx context.Context, target LikeTarget, targetID, userID uint64) error
}

// LikeCountStore 是评论点赞数读取 Port。
type LikeCountStore interface {
	// 批量查询评论的点赞数，返回评论ID到点赞数的映射
	GetCommentLikeCounts(ctx context.Context, commentIDs []uint64) (map[uint64]uint64, error)
}

// HotRankStore 是热榜 Redis Port。
type HotRankStore interface {
	// 查询热度最高的前 limit 个热榜条目
	GetTop(ctx context.Context, limit int) ([]HotRankItem, error)
	// 使用给定条目全量重建热榜
	Rebuild(ctx context.Context, entries []HotRankItem) error
}

// EventPublisher 是 Community 异步事件发布 Port。
type EventPublisher interface {
	// 发布点赞通知事件到消息队列
	SendLikeNotification(ctx context.Context, event NotificationEvent) error
	// 发布浏览历史事件到消息队列
	SendViewHistory(ctx context.Context, event ViewHistoryEvent) error
}
