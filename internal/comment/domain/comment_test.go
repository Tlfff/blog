package domain

import (
	"errors"
	"testing"
)

// TestCommentConstructors 验证主评论和回复构造规则。
func TestCommentConstructors(t *testing.T) {
	root := NewRootComment(1, 2, "主评论", "127.0.0.1")
	if !root.IsRoot() || !root.IsNormal() || root.ReplyToUserID != 0 {
		t.Fatalf("主评论初始化错误: %+v", root)
	}
	reply := NewReply(1, 10, 3, 2, "回复", "127.0.0.1")
	if reply.IsRoot() || !reply.IsNormal() || reply.RootID != 10 || reply.ReplyToUserID != 2 {
		t.Fatalf("回复初始化错误: %+v", reply)
	}
}

// TestCommentReplyAndDeleteRules 验证回复状态和删除权限。
func TestCommentReplyAndDeleteRules(t *testing.T) {
	comment := NewRootComment(1, 2, "评论", "127.0.0.1")
	if err := comment.EnsureReplyable(); err != nil {
		t.Fatalf("正常评论应允许回复: %v", err)
	}
	if err := comment.DeleteBy(3, false); !errors.Is(err, ErrCommentPermission) {
		t.Fatalf("非作者删除错误不正确: %v", err)
	}
	if err := comment.DeleteBy(2, false); err != nil || comment.IsNormal() {
		t.Fatalf("作者删除失败: %v", err)
	}
	if err := comment.EnsureReplyable(); !errors.Is(err, ErrCommentRootDeleted) {
		t.Fatalf("已删除评论应禁止回复: %v", err)
	}
	if err := comment.DeleteBy(2, false); !errors.Is(err, ErrCommentDeleted) {
		t.Fatalf("重复删除错误不正确: %v", err)
	}
}

// TestCommentAdminDeleteAndHotValue 验证管理员删除和热度公式。
func TestCommentAdminDeleteAndHotValue(t *testing.T) {
	comment := &Comment{UserID: 2, Status: CommentStatusNormal, LikeCount: 4, CommentCount: 3}
	if comment.HotValue() != 7 {
		t.Fatalf("评论热度错误: %d", comment.HotValue())
	}
	if err := comment.DeleteBy(99, true); err != nil {
		t.Fatalf("管理员删除失败: %v", err)
	}
}
