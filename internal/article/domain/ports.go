package domain

import (
	"context"
	"time"
)

// ArticleRepository 是文章持久化 Port，由 Infrastructure 层用 MySQL 实现。
type ArticleRepository interface {
	Create(ctx context.Context, article *Article) error                                                            // 新增文章并回填自增ID
	FindByID(ctx context.Context, id uint64) (*Article, error)                                                     // 按ID查询文章聚合根
	FindWithAuthorByID(ctx context.Context, id uint64) (*ArticleWithAuthor, error)                                 // 按ID查询文章并联带作者公开信息
	Update(ctx context.Context, article *Article) error                                                            // 全量更新文章可编辑字段
	SoftDelete(ctx context.Context, articleID uint64) error                                                        // 软删除：仅把状态置为已删除
	Clear(ctx context.Context, articleID uint64) error                                                             // 彻底删除：物理删除数据行
	ListWithCursor(ctx context.Context, lastID uint64, pageSize int, isDesc bool, status int8) ([]*Article, error) // 游标分页查询，适合无限下拉
	ListWithOffset(ctx context.Context, page, pageSize int, isDesc bool, status int8) ([]*Article, error)          // 偏移分页查询，适合带页码的列表
	CountByStatus(ctx context.Context, status int8) (int64, error)                                                 // 按状态统计文章总数
}

// ArticleImageStorage 是文章图片对象存储 Port，由 Infrastructure 层用 MinIO 实现。
type ArticleImageStorage interface {
	PresignedPutURL(ctx context.Context, objectKey string, ttl time.Duration) (string, error) // 生成带有效期的预签名上传URL
	GetObjectURL(publicDomain, objectKey string) string                                       // 拼装对象的对外访问URL
	MoveObject(ctx context.Context, srcKey, dstKey string) error                              // 移动对象，用于草稿图片转正式目录
}

// ArticleInteractionQuery 是文章互动统计查询 Port，由 Community 侧提供。
type ArticleInteractionQuery interface {
	IsUserLikedArticle(ctx context.Context, userID, articleID uint64) (bool, error) // 查询指定用户是否已点赞该文章
}

// UserInfo 是文章详情所需的作者公开信息（跨领域只读查询模型）。
type UserInfo struct {
	ID          uint64 // 用户唯一标识
	Nickname    string // 用户昵称
	Avatar      string // 用户头像URL
	LastLoginIP string // 最后登录IP，用于展示归属地
}

// UserQuery 是作者信息查询 Port，由 Identity 侧提供。
type UserQuery interface {
	FindUserByID(ctx context.Context, id uint64) (*UserInfo, error) // 按用户ID查询作者公开信息
}

// ViewHistoryRepository 定义文章浏览历史持久化能力。
type ViewHistoryRepository interface {
	Create(ctx context.Context, history *ViewHistory) error         // 创建浏览历史
	IncrementViewCount(ctx context.Context, articleID uint64) error // 增加文章浏览量
}

// HotRankStore 定义文章热榜存储能力。
type HotRankStore interface {
	GetTop(ctx context.Context, limit int) ([]HotRankItem, error) // 查询热榜前 N 条
	Rebuild(ctx context.Context, entries []HotRankItem) error     // 全量重建热榜
}

// ArticleInfo 表示热榜查询所需的文章最小信息。
type ArticleInfo struct {
	ID           uint64 // 文章唯一标识
	AuthorID     uint64 // 作者用户唯一标识
	Title        string // 文章标题
	ViewCount    uint32 // 浏览量
	LikeCount    uint32 // 点赞数
	CommentCount uint32 // 评论数
}

// RankingQuery 定义文章热榜所需的数据库查询能力。
type RankingQuery interface {
	GetHotListByIDs(ctx context.Context, ids []uint64) ([]*ArticleInfo, error) // 批量查询热榜文章信息
	GetTopHotArticles(ctx context.Context, limit int) ([]*ArticleInfo, error)  // 查询热度最高的文章
}

// ViewEventPublisher 定义浏览历史事件发布能力。
type ViewEventPublisher interface {
	SendViewHistory(ctx context.Context, event ViewHistoryEvent) error // 发布浏览历史事件
}

// ViewHistoryEvent 表示文章浏览历史领域事件。
type ViewHistoryEvent struct {
	ArticleID   uint64    // 被浏览文章唯一标识
	UserID      uint64    // 浏览用户唯一标识，游客为 0
	CreatedTime time.Time // 浏览发生时间
}

// LikeCountProjection 更新 Article 上下文拥有的点赞数投影。
type LikeCountProjection interface {
	// IncrementLikeCount 按增量调整文章点赞数。
	IncrementLikeCount(ctx context.Context, articleID uint64, delta int64) error
}
