package handler

import (
	"blog/internal/common"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGRPCErrorMappingContract(t *testing.T) {
	tests := []struct {
		err  error
		code codes.Code
	}{
		{common.ErrUserNotFound, codes.NotFound},
		{common.ErrArticleNotFound, codes.NotFound},
		{common.ErrCommentNotFound, codes.NotFound},
		{common.ErrForbidden, codes.PermissionDenied},
		{common.ErrArticlePermissionDenied, codes.PermissionDenied},
		{common.ErrCommentPermission, codes.PermissionDenied},
		{common.ErrTokenInvalid, codes.Unauthenticated},
		{common.ErrTokenExpired, codes.Unauthenticated},
		{common.ErrParameter, codes.InvalidArgument},
		{common.ErrArticleIDInvalid, codes.InvalidArgument},
	}

	for _, tt := range tests {
		st, ok := status.FromError(GRPCError(tt.err))
		if !ok || st.Code() != tt.code {
			t.Errorf("GRPCError(%v) code = %v, want %v", tt.err, st.Code(), tt.code)
		}
	}

	if st, _ := status.FromError(GRPCError(common.ErrSystem)); st.Code() != codes.Internal {
		t.Errorf("未知错误应映射为 Internal, got %v", st.Code())
	}
}
