package repository

import (
	"blog/internal/model"
	"context"

	"gorm.io/gorm"
)

type articleLikeRepository struct {
	db *gorm.DB
}

// GetDB implements [ArticleLikeRepository].
func (r *articleLikeRepository) GetDB() *gorm.DB {
	return r.db
}

type ArticleLikeRepository interface {
	Insert(ctx context.Context, tx *gorm.DB, like *model.ArticleLike) error
	Update(ctx context.Context, tx *gorm.DB, userID, articleID uint64, status int8) error
	UpdateArticleLikeCountDelta(ctx context.Context, tx *gorm.DB, articleID uint64, delta int) error
	FindRecord(ctx context.Context, userID, articleID uint64) (bool, error)
	IsLiked(ctx context.Context, userID, articleID uint64) (bool, error)
	GetLikedUserIDs(ctx context.Context, articleID uint64) ([]uint64, error)

	GetDB() *gorm.DB // 统一事务获取DB句柄，和你CommentRepo规范对齐
}

// 插入点赞记录
// insert into article_likes (user_id,article_id,status) values(?,?,?)
func (r *articleLikeRepository) Insert(ctx context.Context, tx *gorm.DB, like *model.ArticleLike) error {
	return tx.WithContext(ctx).Create(like).Error
}

// 更新评论
// update article_likes set status=? where user_id=? and article_id=?
func (r *articleLikeRepository) Update(ctx context.Context, tx *gorm.DB, userID, articleID uint64, status int8) error {
	return tx.WithContext(ctx).
		Model(&model.ArticleLike{}).
		Where("user_id = ? AND article_id = ?", userID, articleID).
		Update("status", status).Error
}

// 增量更新文章表的like_count（需在事务中调用）
// update articles set like_count = like_count + ? where id = ?
func (r *articleLikeRepository) UpdateArticleLikeCountDelta(ctx context.Context, tx *gorm.DB, articleID uint64, delta int) error {
	return tx.WithContext(ctx).
		Model(&model.Article{}).
		Where("id = ?", articleID).
		UpdateColumn("like_count", gorm.Expr("like_count + ?", delta)).Error
}

// 获取文章点赞状态
// select status from article_likes where user_id=? and article_id=?
func (r *articleLikeRepository) IsLiked(ctx context.Context, userID, articleID uint64) (bool, error) {
	var count int64
	// 查找是否有点赞记录
	err := r.db.WithContext(ctx).
		Model(&model.ArticleLike{}).
		Where("user_id = ? and article_id = ? and status=?", userID, articleID, model.ArticleLiked).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// 获取给某文章点赞的所有用户
// select user_id from article_like where article_id=? and status = ?
func (r *articleLikeRepository) GetLikedUserIDs(ctx context.Context, articleID uint64) ([]uint64, error) {
	var userIDs []uint64
	err := r.db.WithContext(ctx).Model(&model.ArticleLike{}).
		Where("article_id=? and status=?", articleID, model.ArticleLiked).
		Pluck("user_id", &userIDs).Error
	if err != nil {
		return nil, err
	}
	return userIDs, nil
}

// 查找表中是否有该用户操作信息
// select id from artricle_like where user_id=? and article_id =?
func (r *articleLikeRepository) FindRecord(ctx context.Context, userID, articleID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.ArticleLike{}).
		Select("id").
		Where("user_id=? and article_id=?", userID, articleID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
func NewArticleLikeRepository(db *gorm.DB) ArticleLikeRepository {
	return &articleLikeRepository{db: db}
}
