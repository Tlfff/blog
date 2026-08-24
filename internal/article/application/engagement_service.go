package application

import (
	domain "blog/internal/article/domain"
	"log"
)

type EngagementService struct {
	views    domain.ViewHistoryRepository
	hotRank  domain.HotRankStore
	articles domain.RankingQuery
	events   domain.ViewEventPublisher
}

func NewEngagementService(views domain.ViewHistoryRepository, hotRank domain.HotRankStore, articles domain.RankingQuery, events domain.ViewEventPublisher) *EngagementService {
	return &EngagementService{views: views, hotRank: hotRank, articles: articles, events: events}
}

func logError(message string, err error) {
	if err != nil {
		log.Printf("%s: %v", message, err)
	}
}
