package application

import (
	"blog/internal/like/domain"
	"context"
	"testing"
)

type fakeLikeRepository struct{ states map[uint64]bool }

func (f *fakeLikeRepository) SetLiked(_ context.Context, _, targetID uint64, liked bool) error {
	f.states[targetID] = liked
	return nil
}
func (f *fakeLikeRepository) IsLiked(_ context.Context, _, targetID uint64) (bool, error) {
	return f.states[targetID], nil
}
func (f *fakeLikeRepository) GetLikedUserIDs(context.Context, uint64) ([]uint64, error) {
	return nil, nil
}

type fakeTargetQuery struct{}

func (fakeTargetQuery) ArticleExists(context.Context, uint64) (bool, error) { return true, nil }
func (fakeTargetQuery) CommentExists(context.Context, uint64) (bool, error) { return true, nil }

type fakeEventPublisher struct{ created, canceled int }

func (f *fakeEventPublisher) PublishLikeCreated(context.Context, domain.LikeCreated) error {
	f.created++
	return nil
}
func (f *fakeEventPublisher) PublishLikeCanceled(context.Context, domain.LikeCanceled) error {
	f.canceled++
	return nil
}

func TestLikeServiceIsIdempotentAndPublishesEvents(t *testing.T) {
	articles := &fakeLikeRepository{states: map[uint64]bool{}}
	comments := &fakeLikeRepository{states: map[uint64]bool{}}
	events := &fakeEventPublisher{}
	service := NewService(articles, comments, fakeTargetQuery{}, events)
	ctx := context.Background()

	if err := service.LikeArticle(ctx, 1, 10); err != nil {
		t.Fatal(err)
	}
	if err := service.LikeArticle(ctx, 1, 10); err != nil {
		t.Fatal(err)
	}
	if err := service.CancelArticle(ctx, 1, 10); err != nil {
		t.Fatal(err)
	}
	if events.created != 1 || events.canceled != 1 {
		t.Fatalf("事件数量不正确: created=%d canceled=%d", events.created, events.canceled)
	}
}
