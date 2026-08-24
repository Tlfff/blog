package domain

import "testing"

// TestCommentOwnershipAndKind 验证评论归属、主楼和状态规则。
func TestCommentOwnershipAndKind(t *testing.T) {
	comment := &Comment{UserID: 2, RootID: 0, Status: CommentStatusNormal}
	if !comment.IsRoot() || !comment.IsNormal() || !comment.BelongsTo(2) {
		t.Fatalf("评论领域行为错误: %+v", comment)
	}
}
