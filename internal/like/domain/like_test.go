package domain

import "testing"

// TestLikeTargetContract 验证点赞目标和状态常量保持兼容。
func TestLikeTargetContract(t *testing.T) {
	if LikeTargetArticle != "article" || LikeTargetComment != "comment" {
		t.Fatalf("点赞目标常量发生变化: article=%q comment=%q", LikeTargetArticle, LikeTargetComment)
	}
	if LikeStatusLiked != 1 || LikeStatusCanceled != 2 {
		t.Fatalf("点赞状态常量发生变化: liked=%d canceled=%d", LikeStatusLiked, LikeStatusCanceled)
	}
}
