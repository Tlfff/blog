package application

import domain "blog/internal/comment/domain"

type Service struct {
	comments   domain.CommentRepository
	articles   domain.ArticleQuery
	likeCounts domain.LikeCountQuery
	events     domain.CommentEventPublisher
}

// NewService 创建 Comment 上下文应用服务
func NewService(comments domain.CommentRepository, articles domain.ArticleQuery, likeCounts domain.LikeCountQuery, events domain.CommentEventPublisher) *Service {
	return &Service{comments: comments, articles: articles, likeCounts: likeCounts, events: events}
}
