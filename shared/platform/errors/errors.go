// Package errors 提供统一业务错误与 gRPC code 的双向映射。
package errors

import (
	"blog/internal/common"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ToGRPC 把业务错误映射为 gRPC status error。
func ToGRPC(err error) error {
	switch {
	case is(err, common.ErrUserNotFound, common.ErrArticleNotFound, common.ErrCommentNotFound):
		return status.Error(codes.NotFound, err.Error())
	case is(err, common.ErrForbidden, common.ErrArticlePermissionDenied, common.ErrCommentPermission):
		return status.Error(codes.PermissionDenied, err.Error())
	case is(err, common.ErrTokenInvalid, common.ErrTokenExpired, common.ErrTokenSignature, common.ErrTokenIssuer):
		return status.Error(codes.Unauthenticated, err.Error())
	case is(err, common.ErrParameter, common.ErrArticleIDInvalid, common.ErrArticleTitleEmpty, common.ErrArticleContentEmpty, common.ErrArticleStatusInvalid):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, "服务内部错误")
	}
}

// FromGRPC 把 gRPC status error 映射回统一业务错误码。
func FromGRPC(err error) int {
	st, ok := status.FromError(err)
	if !ok {
		return common.CodeInternalServerError
	}
	switch st.Code() {
	case codes.NotFound:
		return common.CodeUserNotFound
	case codes.PermissionDenied:
		return common.CodeForbidden
	case codes.Unauthenticated:
		return common.CodeUnauthorized
	case codes.InvalidArgument:
		return common.CodeInvalidParameter
	default:
		return common.CodeInternalServerError
	}
}

func is(err error, candidates ...error) bool {
	for _, candidate := range candidates {
		if err == candidate {
			return true
		}
	}
	return false
}
