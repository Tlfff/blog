// Package grpc 提供跨上下文共享的 gRPC 协议错误映射。
package grpc

import (
	apperrors "blog/internal/shared/apperrors"
	"errors"
	"log"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Error 将业务错误映射为兼容的 gRPC status error。
func Error(err error) error {
	switch {
	case errors.Is(err, apperrors.ErrUserNotFound),
		errors.Is(err, apperrors.ErrArticleNotFound),
		errors.Is(err, apperrors.ErrCommentNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, apperrors.ErrForbidden),
		errors.Is(err, apperrors.ErrArticlePermissionDenied),
		errors.Is(err, apperrors.ErrCommentPermission):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, apperrors.ErrTokenInvalid),
		errors.Is(err, apperrors.ErrTokenExpired),
		errors.Is(err, apperrors.ErrTokenSignature),
		errors.Is(err, apperrors.ErrTokenIssuer):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, apperrors.ErrParameter),
		errors.Is(err, apperrors.ErrArticleIDInvalid),
		errors.Is(err, apperrors.ErrArticleTitleEmpty),
		errors.Is(err, apperrors.ErrArticleContentEmpty),
		errors.Is(err, apperrors.ErrArticleStatusInvalid):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		log.Printf("[grpc] internal error: %v", err)
		return status.Error(codes.Internal, "服务内部错误")
	}
}
