package community

import (
	domaincommunity "blog/internal/domain/community"
	"blog/internal/model"
	"blog/pkg/database"
	"context"

	"gorm.io/gorm"
)

type articleLikeRepository struct {
	db *gorm.DB
}

// NewArticleLikeRepository 返回直接持有 GORM 的文章点赞 Repository 实现。
func NewArticleLikeRepository(db *gorm.DB) domaincommunity.ArticleLikeRepository {
	return &articleLikeRepository{db: db}
}

func (r *articleLikeRepository) SetLiked(ctx context.Context, userID, articleID uint64, liked bool) error {
	status := domaincommunity.LikeStatusCanceled
	delta := -1
	if liked {
		status = domaincommunity.LikeStatusLiked
		delta = 1
	}
	exists, err := r.findRecord(ctx, userID, articleID)
	if err != nil {
		return err
	}
	return database.RunTx(ctx, r.db, func(tx *gorm.DB) error {
		if exists {
			if err := tx.WithContext(ctx).
				Model(&model.ArticleLike{}).
				Where("user_id = ? AND article_id = ?", userID, articleID).
				Update("status", status).Error; err != nil {
				return err
			}
		} else {
			like := &model.ArticleLike{UserID: userID, ArticleID: articleID, Status: status}
			if err := tx.WithContext(ctx).Create(like).Error; err != nil {
				return err
			}
		}
		return tx.WithContext(ctx).
			Model(&model.Article{}).
			Where("id = ?", articleID).
			UpdateColumn("like_count", gorm.Expr("like_count + ?", delta)).Error
	})
}

func (r *articleLikeRepository) IsLiked(ctx context.Context, userID, articleID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ArticleLike{}).
		Where("user_id = ? and article_id = ? and status=?", userID, articleID, model.ArticleLiked).
		Count(&count).Error
	return count > 0, err
}

func (r *articleLikeRepository) GetLikedUserIDs(ctx context.Context, articleID uint64) ([]uint64, error) {
	var userIDs []uint64
	err := r.db.WithContext(ctx).Model(&model.ArticleLike{}).
		Where("article_id=? and status=?", articleID, model.ArticleLiked).
		Pluck("user_id", &userIDs).Error
	return userIDs, err
}

func (r *articleLikeRepository) findRecord(ctx context.Context, userID, articleID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.ArticleLike{}).
		Select("id").
		Where("user_id=? and article_id=?", userID, articleID).
		Count(&count).Error
	return count > 0, err
}

type commentLikeRepository struct {
	db *gorm.DB
}

// NewCommentLikeRepository 返回直接持有 GORM 的评论点赞 Repository 实现。
func NewCommentLikeRepository(db *gorm.DB) domaincommunity.CommentLikeRepository {
	return &commentLikeRepository{db: db}
}

func (r *commentLikeRepository) SetLiked(ctx context.Context, userID, commentID uint64, liked bool) error {
	status := domaincommunity.LikeStatusCanceled
	delta := -1
	if liked {
		status = domaincommunity.LikeStatusLiked
		delta = 1
	}
	exists, err := r.findRecord(ctx, userID, commentID)
	if err != nil {
		return err
	}
	return database.RunTx(ctx, r.db, func(tx *gorm.DB) error {
		if exists {
			if err := tx.WithContext(ctx).
				Model(&model.CommentLike{}).
				Where("user_id = ? AND comment_id = ?", userID, commentID).
				Update("status", status).Error; err != nil {
				return err
			}
		} else {
			like := &model.CommentLike{UserID: userID, CommentID: commentID, Status: status}
			if err := tx.WithContext(ctx).Create(like).Error; err != nil {
				return err
			}
		}
		return tx.WithContext(ctx).
			Model(&model.Comment{}).
			Where("id = ?", commentID).
			UpdateColumn("like_count", gorm.Expr("like_count + ?", delta)).Error
	})
}

func (r *commentLikeRepository) IsLiked(ctx context.Context, userID, commentID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.CommentLike{}).
		Where("user_id = ? and comment_id = ? and status=?", userID, commentID, model.CommentLiked).
		Count(&count).Error
	return count > 0, err
}

func (r *commentLikeRepository) GetLikedUserIDs(ctx context.Context, commentID uint64) ([]uint64, error) {
	var userIDs []uint64
	err := r.db.WithContext(ctx).Model(&model.CommentLike{}).
		Where("comment_id=? and status=?", commentID, model.CommentLiked).
		Pluck("user_id", &userIDs).Error
	return userIDs, err
}

func (r *commentLikeRepository) findRecord(ctx context.Context, userID, commentID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.CommentLike{}).
		Select("id").
		Where("user_id=? and comment_id=?", userID, commentID).
		Count(&count).Error
	return count > 0, err
}
