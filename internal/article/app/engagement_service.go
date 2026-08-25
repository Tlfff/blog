package app

import (
	domain "blog/internal/article/domain"
	"log"
)

// EngagementService 编排浏览历史和文章热榜用例。
type EngagementService struct {
	views    domain.ViewHistoryRepository // 浏览历史持久化 Port
	hotRank  domain.HotRankStore          // 热榜存储 Port
	articles domain.RankingQuery          // 文章排行查询 Port
	events   domain.ViewEventPublisher    // 浏览历史事件发布 Port
}

// NewEngagementService 创建文章互动应用服务。
func NewEngagementService(views domain.ViewHistoryRepository, hotRank domain.HotRankStore, articles domain.RankingQuery, events domain.ViewEventPublisher) *EngagementService {
	return &EngagementService{views: views, hotRank: hotRank, articles: articles, events: events}
}

// logError 记录不阻断主流程的非致命错误。
func logError(message string, err error) {
	if err != nil {
		log.Printf("%s: %v", message, err)
	}
}
