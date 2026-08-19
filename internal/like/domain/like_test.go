package domain

import (
	"testing"
	"time"
)

func TestLikeLifecycleIsIdempotentAtDomainBoundary(t *testing.T) {
	now := time.Now()
	like := NewLike(1, LikeTargetArticle, 2)
	if err := like.Activate(now); err != nil {
		t.Fatalf("首次点赞失败: %v", err)
	}
	if err := like.Activate(now); err != ErrAlreadyLiked {
		t.Fatalf("重复点赞错误不正确: %v", err)
	}
	if err := like.Cancel(now); err != nil {
		t.Fatalf("取消点赞失败: %v", err)
	}
	if err := like.Cancel(now); err != ErrNotLiked {
		t.Fatalf("重复取消错误不正确: %v", err)
	}
}
