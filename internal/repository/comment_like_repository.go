package repository

import (
	"blog/internal/model"
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CommentLikeRepository interface {
	InsertOrUpdateStatus(ctx context.Context, like *model.CommentLike) error
	Insert(ctx context.Context, like *model.CommentLike) error
	Update(ctx context.Context, userID, commentID uint64, status int8) error
	GetLikeStatus(ctx context.Context, userID, commentID uint64) (int8, error)
	GetLikesByCommentID(ctx context.Context, commentID uint64) ([]*model.CommentLike, error)
	UpdateCommentLikeCountDirectly(ctx context.Context, commentID uint64, count uint32) error
	GetDB() *gorm.DB
}

type commentLikeRepository struct {
	db *gorm.DB
}

func (r *commentLikeRepository) GetDB() *gorm.DB {
	return r.db
}

func NewCommentLikeRepository(db *gorm.DB) CommentLikeRepository {
	return &commentLikeRepository{db: db}
}

// 先插入，有则直接更新
// insert into comment_likes (user_id,comment_id,status) values(?,?,?)
// on duplicate key update status=VALUES(status)
func (r *commentLikeRepository) InsertOrUpdateStatus(ctx context.Context, like *model.CommentLike) error {
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "comment_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"status"}),
		}).
		Create(like).Error
	return err
}

// 插入评论
// insert into comment_likes (user_id,comment_id,status) values(?,?,?)
func (r *commentLikeRepository) Insert(ctx context.Context, like *model.CommentLike) error {
	return r.db.WithContext(ctx).Create(like).Error
}

// 更新评论
// update comment_likes set status=? where user_id=? and comment_id=?
func (r *commentLikeRepository) Update(ctx context.Context, userID, commentID uint64, status int8) error {
	return r.db.WithContext(ctx).
		Model(&model.CommentLike{}).
		Where("user_id = ? AND comment_id = ?", userID, commentID).
		Update("status", status).Error
}

// 获取评论状态
// select status from comment_likes where user_id=? and comment_id=?
func (r *commentLikeRepository) GetLikeStatus(ctx context.Context, userID, commentID uint64) (int8, error) {
	var currentStatus int8
	// 只查询 status 单个字段，提高查询效率
	err := r.db.WithContext(ctx).
		Model(&model.CommentLike{}).
		Select("status").
		Where("user_id = ? AND comment_id = ?", userID, commentID).
		Row().
		Scan(&currentStatus)
	if err != nil {
		return 0, err
	}
	return currentStatus, nil
}

// GetLikesByCommentID 供冷启动提取全量记录使用
func (r *commentLikeRepository) GetLikesByCommentID(ctx context.Context, commentID uint64) ([]*model.CommentLike, error) {
	var list []*model.CommentLike
	err := r.db.WithContext(ctx).
		Where("comment_id = ?", commentID).
		Find(&list).Error
	return list, err
}

// UpdateCommentLikeCountDirectly 定时任务强行同步最新的总点赞数回 comments 评论表
func (r *commentLikeRepository) UpdateCommentLikeCountDirectly(ctx context.Context, commentID uint64, count uint32) error {
	// 假设你的评论表对应的模型是 model.Comment
	return r.db.WithContext(ctx).
		Model(&model.Comment{}).
		Where("id = ?", commentID).
		Update("like_count", count).Error
}
