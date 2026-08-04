package handler

import (
	"blog/internal/common"
	"blog/internal/service"
	"context"
	"errors"

	commentv1 "blog/gen/comment"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// 实现二方服务的评论接口
type CommentHandler struct {
	commentv1.UnimplementedCommentServiceServer
	commentService *service.CommentService
}

func NewCommentHandler(commentService *service.CommentService) *CommentHandler {
	return &CommentHandler{commentService: commentService}
}

// 获取评论的点赞数和热度值
func (h *CommentHandler) GetCommentStats(ctx context.Context, req *commentv1.GetCommentStatsRequest) (*commentv1.GetCommentStatsResponse, error) {
	// 1. 获取评论信息
	stats, err := h.commentService.GetCommentStats(ctx, req.CommentId)
	if err != nil {
		if errors.Is(err, common.ErrCommentNotFound) {
			return nil, status.Error(codes.NotFound, "评论不存在")
		}
		return nil, status.Error(codes.Internal, "服务内部错误")
	}
	// 2. 构建返回响应
	return &commentv1.GetCommentStatsResponse{
		CommentId: stats.ID,
		HotValue:  stats.HotCount,
		LikeCount: stats.LikeCount,
	}, nil
}
