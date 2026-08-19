package application

import (
	"blog/internal/like/domain"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrTargetNotFound = errors.New("点赞目标不存在或不可互动")

type Service struct {
	articles domain.ArticleLikeRepository
	comments domain.CommentLikeRepository
	targets  domain.TargetQuery
	events   domain.EventPublisher
}

func NewService(articles domain.ArticleLikeRepository, comments domain.CommentLikeRepository, targets domain.TargetQuery, events domain.EventPublisher) *Service {
	return &Service{articles: articles, comments: comments, targets: targets, events: events}
}

func (s *Service) LikeArticle(ctx context.Context, userID, articleID uint64) error {
	if s.targets != nil {
		exists, err := s.targets.ArticleExists(ctx, articleID)
		if err != nil {
			return err
		}
		if !exists {
			return ErrTargetNotFound
		}
	}
	liked, err := s.articles.IsLiked(ctx, userID, articleID)
	if err != nil || liked {
		return err
	}
	if err := s.articles.SetLiked(ctx, userID, articleID, true); err != nil {
		return err
	}
	if s.events != nil {
		return s.events.PublishLikeCreated(ctx, domain.LikeCreated{EventID: uuid.NewString(), Version: 1, UserID: userID, Target: domain.LikeTargetArticle, TargetID: articleID, OccurredAt: time.Now()})
	}
	return nil
}

func (s *Service) CancelArticle(ctx context.Context, userID, articleID uint64) error {
	liked, err := s.articles.IsLiked(ctx, userID, articleID)
	if err != nil || !liked {
		return err
	}
	if err := s.articles.SetLiked(ctx, userID, articleID, false); err != nil {
		return err
	}
	if s.events != nil {
		return s.events.PublishLikeCanceled(ctx, domain.LikeCanceled{EventID: uuid.NewString(), Version: 1, UserID: userID, Target: domain.LikeTargetArticle, TargetID: articleID, OccurredAt: time.Now()})
	}
	return nil
}

func (s *Service) LikeComment(ctx context.Context, userID, commentID uint64) error {
	if s.targets != nil {
		exists, err := s.targets.CommentExists(ctx, commentID)
		if err != nil {
			return err
		}
		if !exists {
			return ErrTargetNotFound
		}
	}
	liked, err := s.comments.IsLiked(ctx, userID, commentID)
	if err != nil || liked {
		return err
	}
	if err := s.comments.SetLiked(ctx, userID, commentID, true); err != nil {
		return err
	}
	if s.events != nil {
		return s.events.PublishLikeCreated(ctx, domain.LikeCreated{EventID: uuid.NewString(), Version: 1, UserID: userID, Target: domain.LikeTargetComment, TargetID: commentID, OccurredAt: time.Now()})
	}
	return nil
}

func (s *Service) CancelComment(ctx context.Context, userID, commentID uint64) error {
	liked, err := s.comments.IsLiked(ctx, userID, commentID)
	if err != nil || !liked {
		return err
	}
	if err := s.comments.SetLiked(ctx, userID, commentID, false); err != nil {
		return err
	}
	if s.events != nil {
		return s.events.PublishLikeCanceled(ctx, domain.LikeCanceled{EventID: uuid.NewString(), Version: 1, UserID: userID, Target: domain.LikeTargetComment, TargetID: commentID, OccurredAt: time.Now()})
	}
	return nil
}

func (s *Service) IsUserLikedArticle(ctx context.Context, userID, articleID uint64) (bool, error) {
	if userID == 0 {
		return false, nil
	}
	return s.articles.IsLiked(ctx, userID, articleID)
}
