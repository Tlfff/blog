// Package community 提供 Community 领域的应用用例。
package community

import (
	domaincommunity "blog/internal/domain/community"
)

// Service 编排评论、点赞、浏览、热榜与通知用例。
type Service struct {
	comments      domaincommunity.CommentRepository      // 评论持久化 Port
	articleLikes  domaincommunity.ArticleLikeRepository  // 文章点赞持久化 Port
	commentLikes  domaincommunity.CommentLikeRepository  // 评论点赞持久化 Port
	views         domaincommunity.ViewHistoryRepository  // 浏览历史持久化 Port
	notifications domaincommunity.NotificationRepository // 通知持久化 Port
	articles      domaincommunity.ArticleQuery           // 文章只读查询 Port，由 Content 侧提供
	users         domaincommunity.UserInfoQuery          // 用户只读查询 Port，由 Identity 侧提供
	likeCache     domaincommunity.LikeCache              // 点赞状态缓存 Port
	likeCounts    domaincommunity.LikeCountStore         // 评论点赞数缓存 Port
	hotRank       domaincommunity.HotRankStore           // 热榜 Redis Port
	events        domaincommunity.EventPublisher         // 异步事件发布 Port
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
