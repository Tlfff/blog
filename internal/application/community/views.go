package community

import (
	domaincommunity "blog/internal/domain/community"
	"context"
	"time"
)

// 发送浏览历史事件到消息队列，异步落库以避免阻塞请求
func (s *Service) SendViewHistory(ctx context.Context, userID, articleID uint64) error {
	// 1. 未配置事件发布器时直接跳过，视为浏览历史功能未启用
	if s.events == nil {
		return nil
	}
	// 2. 投递浏览历史事件，携带浏览发生的时间
	return s.events.SendViewHistory(ctx, domaincommunity.ViewHistoryEvent{
		ArticleID:   articleID,
		UserID:      userID,
		CreatedTime: time.Now(),
	})
}

// 消费浏览历史事件：写入浏览记录并自增文章浏览量
func (s *Service) CreateViewHistory(ctx context.Context, userID, articleID uint64, timestamp time.Time) error {
	// 1. 仅登录用户写入浏览历史，游客只累加浏览量
	if userID > 0 {
		history := &domaincommunity.ViewHistory{
			UserID:      userID,
			ArticleID:   articleID,
			CreatedTime: timestamp,
			UpdatedTime: timestamp,
		}
		logError("写入浏览历史失败", s.views.Create(ctx, history))
	}
	// 2. 无论是否登录都自增文章浏览量，失败只记日志不影响主流程
	logError("阅读量自增失败", s.views.IncrementViewCount(ctx, articleID))
	return nil
}
