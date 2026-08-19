package domain

import (
	"errors"
	"time"
)

var (
	ErrAlreadyLiked = errors.New("已经点赞")
	ErrNotLiked     = errors.New("尚未点赞")
)

type Status int8

const (
	StatusLiked    Status = 1
	StatusCanceled Status = 2
)

// Like 是 Like 上下文中的点赞关系聚合，不持有文章或评论实体。
type Like struct {
	UserID     uint64
	Target     LikeTarget
	TargetID   uint64
	Status     Status
	LikedAt    time.Time
	CanceledAt *time.Time
}

func NewLike(userID uint64, target LikeTarget, targetID uint64) *Like {
	return &Like{UserID: userID, Target: target, TargetID: targetID, Status: StatusCanceled}
}

func (like *Like) Activate(now time.Time) error {
	if like.Status == StatusLiked {
		return ErrAlreadyLiked
	}
	like.Status = StatusLiked
	like.LikedAt = now
	like.CanceledAt = nil
	return nil
}

func (like *Like) Cancel(now time.Time) error {
	if like.Status != StatusLiked {
		return ErrNotLiked
	}
	like.Status = StatusCanceled
	like.CanceledAt = &now
	return nil
}

type LikeCreated struct {
	EventID    string
	Version    int
	UserID     uint64
	Target     LikeTarget
	TargetID   uint64
	OccurredAt time.Time
}

type LikeCanceled struct {
	EventID    string
	Version    int
	UserID     uint64
	Target     LikeTarget
	TargetID   uint64
	OccurredAt time.Time
}
