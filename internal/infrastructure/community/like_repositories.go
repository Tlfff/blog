package community

import (
	domaincommunity "blog/internal/domain/community"
	"blog/internal/model"
	"blog/pkg/database"
	"context"

	"gorm.io/gorm"
)

// articleLikeRepository 是文章点赞 Repository 的 GORM 实现。
type articleLikeRepository struct {
	db *gorm.DB // GORM 数据库连接
}

// NewArticleLikeRepository 返回直接持有 GORM 的文章点赞 Repository 实现。
func NewArticleLikeRepository(db *gorm.DB) domaincommunity.ArticleLikeRepository {
	return &articleLikeRepository{db: db}
}

// 设置用户对文章的点赞状态，并同步维护文章点赞计数
func (r *articleLikeRepository) SetLiked(ctx context.Context, userID, articleID uint64, liked bool) error {
	// 1. 根据点赞或取消点赞，确定记录状态与计数增量
	status := domaincommunity.LikeStatusCanceled
	delta := -1
	if liked {
		status = domaincommunity.LikeStatusLiked
		delta = 1
	}
	// 2. 查询点赞记录是否已存在
	exists, err := r.findRecord(ctx, userID, articleID)
	if err != nil {
		return err
	}
	// 3. 在同一事务中写点赞记录并更新文章点赞数
	return database.RunTx(ctx, r.db, func(tx *gorm.DB) error {
		// 3.1 已有记录则更新状态
		if exists {
			if err := tx.WithContext(ctx).
				Model(&model.ArticleLike{}).
				Where("user_id = ? AND article_id = ?", userID, articleID).
				Update("status", status).Error; err != nil {
				return err
			}
		} else {
			// 3.2 无记录则新建一条点赞记录
			like := &model.ArticleLike{UserID: userID, ArticleID: articleID, Status: status}
			if err := tx.WithContext(ctx).Create(like).Error; err != nil {
				return err
			}
		}
		// 3.3 同步增减文章的点赞计数
		return tx.WithContext(ctx).
			Model(&model.Article{}).
			Where("id = ?", articleID).
			UpdateColumn("like_count", gorm.Expr("like_count + ?", delta)).Error
	})
}

// 判断用户当前是否已点赞该文章
func (r *articleLikeRepository) IsLiked(ctx context.Context, userID, articleID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ArticleLike{}).
		Where("user_id = ? and article_id = ? and status=?", userID, articleID, model.ArticleLiked).
		Count(&count).Error
	return count > 0, err
}

// 查询点赞过该文章的全部用户ID
func (r *articleLikeRepository) GetLikedUserIDs(ctx context.Context, articleID uint64) ([]uint64, error) {
	var userIDs []uint64
	err := r.db.WithContext(ctx).Model(&model.ArticleLike{}).
		Where("article_id=? and status=?", articleID, model.ArticleLiked).
		Pluck("user_id", &userIDs).Error
	return userIDs, err
}

// 判断用户与文章的点赞记录是否已存在，决定后续是插入还是更新
func (r *articleLikeRepository) findRecord(ctx context.Context, userID, articleID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.ArticleLike{}).
		Select("id").
		Where("user_id=? and article_id=?", userID, articleID).
		Count(&count).Error
	return count > 0, err
}

// commentLikeRepository 是评论点赞 Repository 的 GORM 实现。
type commentLikeRepository struct {
	db *gorm.DB // GORM 数据库连接
}

// NewCommentLikeRepository 返回直接持有 GORM 的评论点赞 Repository 实现。
func NewCommentLikeRepository(db *gorm.DB) domaincommunity.CommentLikeRepository {
	return &commentLikeRepository{db: db}
}

// 设置用户对评论的点赞状态，并同步维护评论点赞计数
func (r *commentLikeRepository) SetLiked(ctx context.Context, userID, commentID uint64, liked bool) error {
	// 1. 根据点赞或取消点赞，确定记录状态与计数增量
	status := domaincommunity.LikeStatusCanceled
	delta := -1
	if liked {
		status = domaincommunity.LikeStatusLiked
		delta = 1
	}
	// 2. 查询点赞记录是否已存在
	exists, err := r.findRecord(ctx, userID, commentID)
	if err != nil {
		return err
	}
	// 3. 在同一事务中写点赞记录并更新评论点赞数
	return database.RunTx(ctx, r.db, func(tx *gorm.DB) error {
		// 3.1 已有记录则更新状态
		if exists {
			if err := tx.WithContext(ctx).
				Model(&model.CommentLike{}).
				Where("user_id = ? AND comment_id = ?", userID, commentID).
				Update("status", status).Error; err != nil {
				return err
			}
		} else {
			// 3.2 无记录则新建一条点赞记录
			like := &model.CommentLike{UserID: userID, CommentID: commentID, Status: status}
			if err := tx.WithContext(ctx).Create(like).Error; err != nil {
				return err
			}
		}
		// 3.3 同步增减评论的点赞计数
		return tx.WithContext(ctx).
			Model(&model.Comment{}).
			Where("id = ?", commentID).
			UpdateColumn("like_count", gorm.Expr("like_count + ?", delta)).Error
	})
}

// 判断用户当前是否已点赞该评论
func (r *commentLikeRepository) IsLiked(ctx context.Context, userID, commentID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.CommentLike{}).
		Where("user_id = ? and comment_id = ? and status=?", userID, commentID, model.CommentLiked).
		Count(&count).Error
	return count > 0, err
}

// 查询点赞过该评论的全部用户ID
func (r *commentLikeRepository) GetLikedUserIDs(ctx context.Context, commentID uint64) ([]uint64, error) {
	var userIDs []uint64
	err := r.db.WithContext(ctx).Model(&model.CommentLike{}).
		Where("comment_id=? and status=?", commentID, model.CommentLiked).
		Pluck("user_id", &userIDs).Error
	return userIDs, err
}

// 判断用户与评论的点赞记录是否已存在，决定后续是插入还是更新
func (r *commentLikeRepository) findRecord(ctx context.Context, userID, commentID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.CommentLike{}).
		Select("id").
		Where("user_id=? and comment_id=?", userID, commentID).
		Count(&count).Error
	return count > 0, err
}
