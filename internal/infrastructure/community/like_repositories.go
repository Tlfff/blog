package community

import (
	domaincommunity "blog/internal/domain/community"
	"blog/internal/model"
	"blog/internal/repository"
	"blog/pkg/database"
	"context"

	"gorm.io/gorm"
)

type articleLikeRepositoryAdapter struct {
	repo repository.ArticleLikeRepository
}

// NewArticleLikeRepository 将 GORM 文章点赞 Repository 适配为 Community 领域 Port。
func NewArticleLikeRepository(repo repository.ArticleLikeRepository) domaincommunity.ArticleLikeRepository {
	return &articleLikeRepositoryAdapter{repo: repo}
}

func (a *articleLikeRepositoryAdapter) SetLiked(ctx context.Context, userID, articleID uint64, liked bool) error {
	status := domaincommunity.LikeStatusCanceled
	delta := -1
	if liked {
		status = domaincommunity.LikeStatusLiked
		delta = 1
	}
	ok, err := a.repo.FindRecord(ctx, userID, articleID)
	if err != nil {
		return err
	}
	return database.RunTx(ctx, a.repo.GetDB(), func(tx *gorm.DB) error {
		if ok {
			if err := a.repo.Update(ctx, tx, userID, articleID, status); err != nil {
				return err
			}
		} else {
			like := &model.ArticleLike{UserID: userID, ArticleID: articleID, Status: status}
			if err := a.repo.Insert(ctx, tx, like); err != nil {
				return err
			}
		}
		return a.repo.UpdateArticleLikeCountDelta(ctx, tx, articleID, delta)
	})
}

func (a *articleLikeRepositoryAdapter) IsLiked(ctx context.Context, userID, articleID uint64) (bool, error) {
	return a.repo.IsLiked(ctx, userID, articleID)
}

func (a *articleLikeRepositoryAdapter) GetLikedUserIDs(ctx context.Context, articleID uint64) ([]uint64, error) {
	return a.repo.GetLikedUserIDs(ctx, articleID)
}

type commentLikeRepositoryAdapter struct {
	repo repository.CommentLikeRepository
}

// NewCommentLikeRepository 将 GORM 评论点赞 Repository 适配为 Community 领域 Port。
func NewCommentLikeRepository(repo repository.CommentLikeRepository) domaincommunity.CommentLikeRepository {
	return &commentLikeRepositoryAdapter{repo: repo}
}

func (a *commentLikeRepositoryAdapter) SetLiked(ctx context.Context, userID, commentID uint64, liked bool) error {
	status := domaincommunity.LikeStatusCanceled
	delta := -1
	if liked {
		status = domaincommunity.LikeStatusLiked
		delta = 1
	}
	ok, err := a.repo.FindRecord(ctx, userID, commentID)
	if err != nil {
		return err
	}
	return database.RunTx(ctx, a.repo.GetDB(), func(tx *gorm.DB) error {
		if ok {
			if err := a.repo.Update(ctx, tx, userID, commentID, status); err != nil {
				return err
			}
		} else {
			like := &model.CommentLike{UserID: userID, CommentID: commentID, Status: status}
			if err := a.repo.Insert(ctx, tx, like); err != nil {
				return err
			}
		}
		return a.repo.UpdateCommentLikeCountDelta(ctx, tx, commentID, delta)
	})
}

func (a *commentLikeRepositoryAdapter) IsLiked(ctx context.Context, userID, commentID uint64) (bool, error) {
	return a.repo.IsLiked(ctx, userID, commentID)
}

func (a *commentLikeRepositoryAdapter) GetLikedUserIDs(ctx context.Context, commentID uint64) ([]uint64, error) {
	return a.repo.GetLikedUserIDs(ctx, commentID)
}
