package infrastructure

import (
	"blog/internal/like/domain"
	"blog/internal/like/infrastructure/model"
	platformtransaction "blog/internal/platform/transaction"
	"context"
	"errors"

	"gorm.io/gorm"
)

// articleLikeRepository 是文章点赞 Repository 的 GORM 实现。
type articleLikeRepository struct {
	db *gorm.DB // 默认 GORM 数据库连接
}

// NewArticleLikeRepository 创建文章点赞 Repository。
func NewArticleLikeRepository(db *gorm.DB) domain.ArticleLikeRepository {
	return &articleLikeRepository{db: db}
}

// SetLiked 更新或创建文章点赞关系，并返回状态是否发生变化。
func (r *articleLikeRepository) SetLiked(ctx context.Context, userID, articleID uint64, liked bool) (bool, error) {
	db := platformtransaction.DB(ctx, r.db)
	var record model.ArticleLike
	err := db.WithContext(ctx).Where("user_id = ? AND article_id = ?", userID, articleID).Take(&record).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}
	targetStatus := int8(model.ArticleCancelLiked)
	if liked {
		targetStatus = model.ArticleLiked
	}
	if err == nil && record.Status == targetStatus {
		return false, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		result := db.WithContext(ctx).Create(&model.ArticleLike{UserID: userID, ArticleID: articleID, Status: targetStatus})
		return result.RowsAffected > 0, result.Error
	}
	result := db.WithContext(ctx).Model(&model.ArticleLike{}).
		Where("user_id = ? AND article_id = ?", userID, articleID).
		Update("status", targetStatus)
	return result.RowsAffected > 0, result.Error
}

// IsLiked 判断用户当前是否点赞文章。
func (r *articleLikeRepository) IsLiked(ctx context.Context, userID, articleID uint64) (bool, error) {
	var count int64
	err := platformtransaction.DB(ctx, r.db).WithContext(ctx).
		Model(&model.ArticleLike{}).
		Where("user_id = ? AND article_id = ? AND status = ?", userID, articleID, model.ArticleLiked).
		Count(&count).Error
	return count > 0, err
}

// GetLikedUserIDs 查询点赞文章的全部用户唯一标识。
func (r *articleLikeRepository) GetLikedUserIDs(ctx context.Context, articleID uint64) ([]uint64, error) {
	var userIDs []uint64
	err := platformtransaction.DB(ctx, r.db).WithContext(ctx).Model(&model.ArticleLike{}).
		Where("article_id = ? AND status = ?", articleID, model.ArticleLiked).
		Pluck("user_id", &userIDs).Error
	return userIDs, err
}

// commentLikeRepository 是评论点赞 Repository 的 GORM 实现。
type commentLikeRepository struct {
	db *gorm.DB // 默认 GORM 数据库连接
}

// NewCommentLikeRepository 创建评论点赞 Repository。
func NewCommentLikeRepository(db *gorm.DB) domain.CommentLikeRepository {
	return &commentLikeRepository{db: db}
}

// SetLiked 更新或创建评论点赞关系，并返回状态是否发生变化。
func (r *commentLikeRepository) SetLiked(ctx context.Context, userID, commentID uint64, liked bool) (bool, error) {
	db := platformtransaction.DB(ctx, r.db)
	var record model.CommentLike
	err := db.WithContext(ctx).Where("user_id = ? AND comment_id = ?", userID, commentID).Take(&record).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}
	targetStatus := int8(model.CommentCancelLiked)
	if liked {
		targetStatus = model.CommentLiked
	}
	if err == nil && record.Status == targetStatus {
		return false, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		result := db.WithContext(ctx).Create(&model.CommentLike{UserID: userID, CommentID: commentID, Status: targetStatus})
		return result.RowsAffected > 0, result.Error
	}
	result := db.WithContext(ctx).Model(&model.CommentLike{}).
		Where("user_id = ? AND comment_id = ?", userID, commentID).
		Update("status", targetStatus)
	return result.RowsAffected > 0, result.Error
}

// IsLiked 判断用户当前是否点赞评论。
func (r *commentLikeRepository) IsLiked(ctx context.Context, userID, commentID uint64) (bool, error) {
	var count int64
	err := platformtransaction.DB(ctx, r.db).WithContext(ctx).
		Model(&model.CommentLike{}).
		Where("user_id = ? AND comment_id = ? AND status = ?", userID, commentID, model.CommentLiked).
		Count(&count).Error
	return count > 0, err
}

// GetLikedUserIDs 查询点赞评论的全部用户唯一标识。
func (r *commentLikeRepository) GetLikedUserIDs(ctx context.Context, commentID uint64) ([]uint64, error) {
	var userIDs []uint64
	err := platformtransaction.DB(ctx, r.db).WithContext(ctx).Model(&model.CommentLike{}).
		Where("comment_id = ? AND status = ?", commentID, model.CommentLiked).
		Pluck("user_id", &userIDs).Error
	return userIDs, err
}
