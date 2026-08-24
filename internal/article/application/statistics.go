package application

import (
	"blog/internal/article/domain"
	"context"
)

// StatisticsService 提供文章互动统计的本地 Application Facade。
type StatisticsService struct {
	writer domain.StatisticsWriter // Article 统计写入 Port
}

// NewStatisticsService 创建文章互动统计应用服务。
func NewStatisticsService(writer domain.StatisticsWriter) *StatisticsService {
	return &StatisticsService{writer: writer}
}

// IncrementCommentCount 按增量调整文章评论数。
func (s *StatisticsService) IncrementCommentCount(ctx context.Context, articleID uint64, delta int64) error {
	return s.writer.IncrementCommentCount(ctx, articleID, delta)
}

// AdjustLikeCount 按增量调整文章点赞数。
func (s *StatisticsService) AdjustLikeCount(ctx context.Context, articleID uint64, delta int64) error {
	return s.writer.IncrementLikeCount(ctx, articleID, delta)
}
