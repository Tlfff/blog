package application

import (
	"blog/internal/like/domain"
	"context"
	"testing"
)

// fakeLikeRepository 是文章和评论点赞关系的内存实现。
type fakeLikeRepository struct {
	liked map[uint64]map[uint64]bool // targetID -> userID -> liked
}

// newFakeLikeRepository 创建内存点赞 Repository。
func newFakeLikeRepository() *fakeLikeRepository {
	return &fakeLikeRepository{liked: make(map[uint64]map[uint64]bool)}
}

// SetLiked 更新点赞状态并返回状态是否变化。
func (f *fakeLikeRepository) SetLiked(_ context.Context, userID, targetID uint64, liked bool) (bool, error) {
	if f.liked[targetID] == nil {
		f.liked[targetID] = make(map[uint64]bool)
	}
	if f.liked[targetID][userID] == liked {
		return false, nil
	}
	f.liked[targetID][userID] = liked
	return true, nil
}

// IsLiked 查询点赞状态。
func (f *fakeLikeRepository) IsLiked(_ context.Context, userID, targetID uint64) (bool, error) {
	return f.liked[targetID][userID], nil
}

// GetLikedUserIDs 查询点赞用户列表。
func (f *fakeLikeRepository) GetLikedUserIDs(_ context.Context, targetID uint64) ([]uint64, error) {
	var ids []uint64
	for id, liked := range f.liked[targetID] {
		if liked {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// fakeCache 是点赞状态缓存的内存实现。
type fakeCache struct {
	repo *fakeLikeRepository // 回源 Repository
}

// IsLiked 查询点赞状态。
func (f fakeCache) IsLiked(ctx context.Context, _ domain.LikeTarget, targetID, userID uint64) (bool, error) {
	return f.repo.IsLiked(ctx, userID, targetID)
}

// Add 将用户加入点赞集合。
func (fakeCache) Add(context.Context, domain.LikeTarget, uint64, uint64) error { return nil }

// Remove 将用户移出点赞集合。
func (fakeCache) Remove(context.Context, domain.LikeTarget, uint64, uint64) error { return nil }

// fakeProjection 记录点赞数增量。
type fakeProjection struct {
	deltas []int64 // 收到的点赞数增量
}

// ApplyLikeDelta 记录点赞数增量。
func (f *fakeProjection) ApplyLikeDelta(_ context.Context, _ domain.LikeTarget, _ uint64, delta int64) error {
	f.deltas = append(f.deltas, delta)
	return nil
}

// fakeEventPublisher 记录文章点赞通知次数。
type fakeEventPublisher struct {
	created int // 通知次数
}

// PublishLikeCreated 记录文章点赞通知。
func (f *fakeEventPublisher) PublishLikeCreated(context.Context, domain.LikeCreated) error {
	f.created++
	return nil
}

// fakeTransactionManager 直接执行事务回调。
type fakeTransactionManager struct{}

// WithinTransaction 执行事务回调。
func (fakeTransactionManager) WithinTransaction(ctx context.Context, callback func(context.Context) error) error {
	return callback(ctx)
}

// TestArticleLikeLifecycle 验证文章点赞幂等、统计和通知行为。
func TestArticleLikeLifecycle(t *testing.T) {
	repo := newFakeLikeRepository()
	projection := &fakeProjection{}
	events := &fakeEventPublisher{}
	service := NewService(repo, newFakeLikeRepository(), fakeCache{repo: repo}, events, projection, fakeTransactionManager{})

	if err := service.ArticleLike(context.Background(), 1, 10); err != nil {
		t.Fatalf("文章点赞失败: %v", err)
	}
	if err := service.ArticleLike(context.Background(), 1, 10); err != nil {
		t.Fatalf("重复文章点赞应成功: %v", err)
	}
	if events.created != 1 {
		t.Fatalf("重复点赞不应重复发布通知: %d", events.created)
	}
	if err := service.ArticleCancelLike(context.Background(), 1, 10); err != nil {
		t.Fatalf("取消文章点赞失败: %v", err)
	}
	if len(projection.deltas) != 2 || projection.deltas[0] != 1 || projection.deltas[1] != -1 {
		t.Fatalf("点赞统计增量不正确: %v", projection.deltas)
	}
}
