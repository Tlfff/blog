// Package community 提供 Community 领域的应用用例。
package community

import (
	domaincommunity "blog/internal/domain/community"
)

// Service 编排评论、点赞、浏览、热榜与通知用例。
type Service struct {
	comments      domaincommunity.CommentRepository
	articleLikes  domaincommunity.ArticleLikeRepository
	commentLikes  domaincommunity.CommentLikeRepository
	views         domaincommunity.ViewHistoryRepository
	notifications domaincommunity.NotificationRepository
	articles      domaincommunity.ArticleQuery
	users         domaincommunity.UserInfoQuery
	likeCache     domaincommunity.LikeCache
	likeCounts    domaincommunity.LikeCountStore
	hotRank       domaincommunity.HotRankStore
	events        domaincommunity.EventPublisher
}

// NewService 组装 Community Application 用例依赖。
func NewService(
	comments domaincommunity.CommentRepository,
	articleLikes domaincommunity.ArticleLikeRepository,
	commentLikes domaincommunity.CommentLikeRepository,
	views domaincommunity.ViewHistoryRepository,
	notifications domaincommunity.NotificationRepository,
	articles domaincommunity.ArticleQuery,
	users domaincommunity.UserInfoQuery,
	likeCache domaincommunity.LikeCache,
	likeCounts domaincommunity.LikeCountStore,
	hotRank domaincommunity.HotRankStore,
	events domaincommunity.EventPublisher,
) *Service {
	return &Service{
		comments:      comments,
		articleLikes:  articleLikes,
		commentLikes:  commentLikes,
		views:         views,
		notifications: notifications,
		articles:      articles,
		users:         users,
		likeCache:     likeCache,
		likeCounts:    likeCounts,
		hotRank:       hotRank,
		events:        events,
	}
}
