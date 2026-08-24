package app

import (
	"blog/internal/comment/domain"
	"context"
)

// LikeProjectionService 更新 Comment 上下文拥有的点赞数投影。
type LikeProjectionService struct {
	projection domain.LikeCountProjection // 评论点赞数投影 Port
}

// NewLikeProjectionService 创建评论点赞数投影服务。
func NewLikeProjectionService(projection domain.LikeCountProjection) *LikeProjectionService {
	return &LikeProjectionService{projection: projection}
}

// AdjustLikeCount 调整评论点赞数投影。
func (s *LikeProjectionService) AdjustLikeCount(ctx context.Context, commentID uint64, delta int64) error {
	return s.projection.IncrementLikeCount(ctx, commentID, delta)
}
