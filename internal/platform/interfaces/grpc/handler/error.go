package handler

import (
	"blog/internal/shared/common"
	"errors"
	"log"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCError 把业务层错误统一映射为 gRPC status error。
// 已知业务错误向调用方暴露具体原因；未知错误记录完整日志，对外只返回通用消息。
func GRPCError(err error) error {
	switch {
	// 资源不存在
	case errors.Is(err, common.ErrUserNotFound),
		errors.Is(err, common.ErrArticleNotFound),
		errors.Is(err, common.ErrCommentNotFound):
		return status.Error(codes.NotFound, err.Error())

	// 权限不足
	case errors.Is(err, common.ErrForbidden),
		errors.Is(err, common.ErrArticlePermissionDenied),
		errors.Is(err, common.ErrCommentPermission):
		return status.Error(codes.PermissionDenied, err.Error())

	// 认证失败
	case errors.Is(err, common.ErrTokenInvalid),
		errors.Is(err, common.ErrTokenExpired),
		errors.Is(err, common.ErrTokenSignature),
		errors.Is(err, common.ErrTokenIssuer):
		return status.Error(codes.Unauthenticated, err.Error())

	// 参数错误
	case errors.Is(err, common.ErrParameter),
		errors.Is(err, common.ErrArticleIDInvalid),
		errors.Is(err, common.ErrArticleTitleEmpty),
		errors.Is(err, common.ErrArticleContentEmpty),
		errors.Is(err, common.ErrArticleStatusInvalid):
		return status.Error(codes.InvalidArgument, err.Error())

	default:
		// 未知错误：记录完整信息便于排查，对外隐藏内部细节
		log.Printf("[grpc] internal error: %v", err)
		return status.Error(codes.Internal, "服务内部错误")
	}
}
