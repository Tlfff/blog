package community

import "testing"

func TestCommentHierarchyAndPermissionRules(t *testing.T) {
	root := &Comment{ID: 1, UserID: 10, RootID: 0, Status: CommentStatusNormal}
	reply := &Comment{ID: 2, UserID: 11, RootID: 1, Status: CommentStatusNormal}

	if !root.IsRoot() || reply.IsRoot() {
		t.Fatal("评论层级规则错误")
	}
	if !root.BelongsTo(10) || root.BelongsTo(11) {
		t.Fatal("评论归属规则错误")
	}
	if !root.IsNormal() {
		t.Fatal("正常评论状态规则错误")
	}
	root.Status = CommentStatusDeleted
	if root.IsNormal() {
		t.Fatal("删除后评论不应为正常状态")
	}
}

func TestLikeIdempotencyState(t *testing.T) {
	if LikeStatusLiked != 1 || LikeStatusCanceled != 2 {
		t.Fatalf("点赞状态常量被改变: %d %d", LikeStatusLiked, LikeStatusCanceled)
	}
}

func TestHotScoreFormula(t *testing.T) {
	score := CalcHotScore(10, 5, 3)
	if score != float64(10+5+3) {
		t.Fatalf("热榜公式被改变: %v", score)
	}
}
