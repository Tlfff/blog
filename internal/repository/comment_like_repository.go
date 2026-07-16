package repository

import (
	"blog/internal/model"
	"context"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CommentLikeRepository interface {
	InsertOrUpdateStatus(ctx context.Context, like *model.CommentLike) error
	Insert(ctx context.Context, like *model.CommentLike) error
	Update(ctx context.Context, userID, commentID uint64, status int8) error
	GetLikeStatus(ctx context.Context, userID, commentID uint64) (int8, error)
	GetLikesByID(ctx context.Context, commentID uint64) ([]*model.CommentLike, error)
	UpdateCommentLikeCountDirectly(ctx context.Context, commentID uint64, count uint32) error
	CountValidByCommentID(ctx context.Context, commentID uint64) (int64, error)
	UpdateCommentLikeCount(ctx context.Context, commentID uint64, count int64) error
	BatchInsertOrUpdateStatus(ctx context.Context, likes []*model.CommentLike) error
	BatchUpdateCommentLikeCount(ctx context.Context, comments []*model.Comment) error

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

// 获取点赞评论数据
func (r *commentLikeRepository) GetLikesByID(ctx context.Context, commentID uint64) ([]*model.CommentLike, error) {
	var list []*model.CommentLike
	err := r.db.WithContext(ctx).
		Select("user_id", "status").
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

// UpdateCommentLikeCount 定时任务同步评论点赞总数（int64版本）
func (r *commentLikeRepository) UpdateCommentLikeCount(ctx context.Context, commentID uint64, count int64) error {
	return r.db.WithContext(ctx).
		Model(&model.Comment{}).
		Where("id = ?", commentID).
		Update("like_count", count).Error
}

// 统计有效点赞总数 status=1
func (r *commentLikeRepository) CountValidByCommentID(ctx context.Context, commentID uint64) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&model.CommentLike{}).
		Where("comment_id = ? AND status = ?", commentID, model.CommentLiked).
		Count(&total).Error
	return total, err
}

// 评论插入，如果有则更新，批量 upsert
// insert into comment_likes (user_id,comment_id,status) values(?,?,?)
// on duplicate key update status=VALUES(status)
func (r *commentLikeRepository) BatchInsertOrUpdateStatus(ctx context.Context, likes []*model.CommentLike) error {
	err := r.db.WithContext(ctx).
		Model(&model.CommentLike{}).
		Select("user_id", "comment_id", "status").
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "comment_id"}}, // 唯一索引列
			DoUpdates: clause.AssignmentColumns([]string{"status"}),             // 发生冲突时仅更新点赞状态字段
		}).
		CreateInBatches(likes, 100).Error // 100 条打包一组批量SQL

	return err
}

// 批量更新评论主表的点赞字段
//
//	UPDATE comments
//	SET like_count = CASE WHEN id=? THEN ? WHEN id=? THEN ? ... END
//	WHERE id IN (?, ?, ...)
func (r *commentLikeRepository) BatchUpdateCommentLikeCount(ctx context.Context, comments []*model.Comment) error {
	var caseSql string     // 拼接 case when 分支语句
	var ids []uint64       // 存储所有待更新评论id
	var args []interface{} // SQL参数数组，防止SQL注入

	// 循环构造每条记录的case匹配规则
	for _, comment := range comments {
		caseSql += " WHEN id = ? THEN ? "
		ids = append(ids, comment.ID)
		args = append(args, comment.ID, comment.LikeCount)
	}

	// 构造完整SQL
	sql := "UPDATE comments SET like_count = CASE " + caseSql + " END WHERE id IN (?" + strings.Repeat(",?", len(ids)-1) + ")"
	// 把id列表转为参数数组，拼接到末尾WHERE条件
	idArgs := make([]interface{}, len(ids))
	for i, v := range ids {
		idArgs[i] = v
	}
	args = append(args, idArgs...)

	// 执行原生SQL
	return r.db.WithContext(ctx).Exec(sql, args...).Error
}
