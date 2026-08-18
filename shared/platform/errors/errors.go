// Package errors 提供统一业务错误与 gRPC code 的双向映射。
package errors

import (
	"blog/internal/common"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// 把业务错误映射为 gRPC status error，供内部服务端返回
func ToGRPC(err error) error {
	// 1. 按错误类别匹配到对应的 gRPC code
	switch {
	case is(err, common.ErrUserNotFound, common.ErrArticleNotFound, common.ErrCommentNotFound):
		return status.Error(codes.NotFound, err.Error())
	case is(err, common.ErrForbidden, common.ErrArticlePermissionDenied, common.ErrCommentPermission):
		return status.Error(codes.PermissionDenied, err.Error())
	case is(err, common.ErrTokenInvalid, common.ErrTokenExpired, common.ErrTokenSignature, common.ErrTokenIssuer):
		return status.Error(codes.Unauthenticated, err.Error())
	case is(err, common.ErrParameter, common.ErrArticleIDInvalid, common.ErrArticleTitleEmpty, common.ErrArticleContentEmpty, common.ErrArticleStatusInvalid):
		return status.Error(codes.InvalidArgument, err.Error())
	// 2. 未识别的错误统一返回 Internal，避免泄漏内部细节
	default:
		return status.Error(codes.Internal, "服务内部错误")
	}
}

// 把 gRPC status error 映射回统一业务错误码，供网关侧使用
func FromGRPC(err error) int {
	// 1. 非 gRPC status error 统一按内部错误处理
	st, ok := status.FromError(err)
	if !ok {
		return common.CodeInternalServerError
	}
	// 2. 按 gRPC code 反向映射为业务错误码
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

// // 判断错误是否属于给定候选错误之一
func is(err error, candidates ...error) bool {
	for _, candidate := range candidates {
		if err == candidate {
			return true
		}
	}
	return false
}
