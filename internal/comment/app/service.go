package app

import commentdomain "blog/internal/comment/domain"

// Service 编排 Comment 上下文用例。
type Service struct {
	comments   commentdomain.CommentRepository // 评论持久化 Port
	likeCounts commentdomain.LikeCountQuery    // 评论点赞数查询 Port
	tx         TransactionManager              // 本地事务协调 Port
}

// NewService 创建 Comment 上下文应用服务。
func NewService(comments commentdomain.CommentRepository, likeCounts commentdomain.LikeCountQuery, tx TransactionManager) *Service {
	return &Service{comments: comments, likeCounts: likeCounts, tx: tx}
}

// InteractionTarget 表示跨上下文查询所需的最小评论信息。
type InteractionTarget struct {
	ID        uint64 // 评论唯一标识
	ArticleID uint64 // 所属文章唯一标识
	UserID    uint64 // 评论作者用户唯一标识
	Active    bool   // 评论是否可正常读取
}
