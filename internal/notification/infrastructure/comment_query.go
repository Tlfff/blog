package infrastructure

import (
	commentdomain "blog/internal/comment/domain"
	domain "blog/internal/notification/domain"
	"context"
)

// commentQuery 是 Notification 到 Comment 的防腐查询适配器
type commentQuery struct {
	repository commentdomain.CommentRepository // Comment 上下文查询 Port
}

// 创建评论查询适配器
func NewCommentQuery(repository commentdomain.CommentRepository) domain.CommentQuery {
	return &commentQuery{repository: repository}
}

// 查询通知目标评论的必要信息
func (q *commentQuery) FindByID(ctx context.Context, id uint64) (*domain.CommentInfo, error) {
	comment, err := q.repository.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &domain.CommentInfo{ID: comment.ID, ArticleID: comment.ArticleID, UserID: comment.UserID}, nil
}
