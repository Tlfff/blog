package community

import (
	domaincommunity "blog/internal/domain/community"
	"context"
	"time"
)

func (s *Service) SendViewHistory(ctx context.Context, userID, articleID uint64) error {
	if s.events == nil {
		return nil
	}
	return s.events.SendViewHistory(ctx, domaincommunity.ViewHistoryEvent{
		ArticleID:   articleID,
		UserID:      userID,
		CreatedTime: time.Now(),
	})
}

func (s *Service) CreateViewHistory(ctx context.Context, userID, articleID uint64, timestamp time.Time) error {
	if userID > 0 {
		history := &domaincommunity.ViewHistory{
			UserID:      userID,
			ArticleID:   articleID,
			CreatedTime: timestamp,
			UpdatedTime: timestamp,
		}
		logError("写入浏览历史失败", s.views.Create(ctx, history))
	}
	logError("阅读量自增失败", s.views.IncrementViewCount(ctx, articleID))
	return nil
}
