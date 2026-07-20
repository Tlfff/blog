package repository

import (
	"blog/internal/model"
	"context"

	"gorm.io/gorm"
)

type CommentLikeRepository interface {
	Insert(ctx context.Context, tx *gorm.DB, like *model.CommentLike) error
	Update(ctx context.Context, tx *gorm.DB, userID, commentID uint64, status int8) error
	UpdateCommentLikeCountDelta(ctx context.Context, tx *gorm.DB, commentID uint64, delta int) error
	FindRecord(ctx context.Context, userID, commentID uint64) (bool, error)
	IsLiked(ctx context.Context, userID, commentID uint64) (bool, error)
	GetLikedUserIDs(ctx context.Context, commentID uint64) ([]uint64, error)

	GetDB() *gorm.DB
}

type commentLikeRepository struct {
	db *gorm.DB
}

//	增量更新评论表的like_count
//
// update comments set like_count = like_count + ? where id = ?
func (r *commentLikeRepository) UpdateCommentLikeCountDelta(ctx context.Context, tx *gorm.DB, commentID uint64, delta int) error {
	return tx.WithContext(ctx).
		Model(&model.Comment{}).
		Where("id = ?", commentID).
		UpdateColumn("like_count", gorm.Expr("like_count + ?", delta)).Error
}

// 查找表中是否有该用户操作信息
// select id from comment_like where user_id=? and comment_id =?
func (r *commentLikeRepository) FindRecord(ctx context.Context, userID uint64, commentID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.CommentLike{}).
		Select("id").
		Where("user_id=? and comment_id=?", userID, commentID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// 获取给某文章点赞的所有用户
// select user_id from comment_like where comment_id=? and status = ?
func (r *commentLikeRepository) GetLikedUserIDs(ctx context.Context, commentID uint64) ([]uint64, error) {
	var userIDs []uint64
	err := r.db.WithContext(ctx).Model(&model.CommentLike{}).
		Where("comment_id=? and status=?", commentID, model.CommentLiked).
		Pluck("user_id", &userIDs).Error
	if err != nil {
		return nil, err
	}
	return userIDs, nil
}

// 获取评论点赞状态
// select status from comment_like where user_id=? and comment_id=?
func (r *commentLikeRepository) IsLiked(ctx context.Context, userID uint64, commentID uint64) (bool, error) {
	var count int64
	// 查找是否有点赞记录
	err := r.db.WithContext(ctx).
		Model(&model.CommentLike{}).
		Where("user_id = ? and comment_id = ? and status=?", userID, commentID, model.CommentLiked).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *commentLikeRepository) GetDB() *gorm.DB {
	return r.db
}

// 插入评论
// insert into comment_likes (user_id,comment_id,status) values(?,?,?)
func (r *commentLikeRepository) Insert(ctx context.Context, tx *gorm.DB, like *model.CommentLike) error {
	return tx.WithContext(ctx).Create(like).Error
}

// 更新评论
// update comment_likes set status=? where user_id=? and comment_id=?
func (r *commentLikeRepository) Update(ctx context.Context, tx *gorm.DB, userID, commentID uint64, status int8) error {
	return tx.WithContext(ctx).
		Model(&model.CommentLike{}).
		Where("user_id = ? AND comment_id = ?", userID, commentID).
		Update("status", status).Error
}

func NewCommentLikeRepository(db *gorm.DB) CommentLikeRepository {
	return &commentLikeRepository{db: db}
}
