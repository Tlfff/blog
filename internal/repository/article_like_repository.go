package repository

import (
	"blog/internal/model"
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type articleLikeRepository struct {
	db *gorm.DB
}

// GetDB implements [ArticleLikeRepository].
func (r *articleLikeRepository) GetDB() *gorm.DB {
	return r.db
}

type ArticleLikeRepository interface {
	InsertOrUpdateStatus(ctx context.Context, like *model.ArticleLike) error
	Insert(ctx context.Context, like *model.ArticleLike) error
	Update(ctx context.Context, userID, articleID uint64, status int8) error
	GetLikeStatus(ctx context.Context, userID, articleID uint64) (int8, error)
	GetLikesByArticleID(ctx context.Context, articleID uint64) ([]*model.ArticleLike, error) // <-- 新增此行
	UpdateArticleLikeCountDirectly(ctx context.Context, articleID uint64, count uint32) error
	GetDB() *gorm.DB // 统一事务获取DB句柄，和你CommentRepo规范对齐
}

func NewArticleLikeRepository(db *gorm.DB) ArticleLikeRepository {
	return &articleLikeRepository{db: db}
}

// 先插入，有则直接更新
// insert into article_likes (user_id,article_id,status) values(?,?,?)
// on duplicate key update status=VALUES(status)
func (r *articleLikeRepository) InsertOrUpdateStatus(ctx context.Context, like *model.ArticleLike) error {
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "article_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"status"}),
		}).
		Create(like).Error
	return err
}

// 插入点赞记录
// insert into article_likes (user_id,article_id,status) values(?,?,?)
func (r *articleLikeRepository) Insert(ctx context.Context, like *model.ArticleLike) error {
	return r.db.WithContext(ctx).Create(like).Error
}

//	获取某篇文章的所有点赞记录（包含点赞和取消点赞）
//
// select id, user_id, article_id, status from article_likes where article_id=?
func (r *articleLikeRepository) GetLikesByArticleID(ctx context.Context, articleID uint64) ([]*model.ArticleLike, error) {
	var list []*model.ArticleLike
	err := r.db.WithContext(ctx).
		Model(&model.ArticleLike{}).
		Select(`user_id,article_id,status`).
		Where("article_id = ?", articleID).
		Find(&list).Error
	return list, err
}

// 更新评论
// update article_likes set status=? where user_id=? and article_id=?
func (r *articleLikeRepository) Update(ctx context.Context, userID, articleID uint64, status int8) error {
	return r.db.WithContext(ctx).
		Model(&model.ArticleLike{}).
		Where("user_id = ? AND article_id = ?", userID, articleID).
		Update("status", status).Error
}

// 获取评论状态
// select status from article_likes where user_id=? and article_id=?
func (r *articleLikeRepository) GetLikeStatus(ctx context.Context, userID, articleID uint64) (int8, error) {
	var currentStatus int8
	// 只查询 status 单个字段，提高查询效率
	err := r.db.WithContext(ctx).
		Model(&model.ArticleLike{}).
		Select("status").
		Where("user_id = ? AND article_id = ?", userID, articleID).
		Row().
		Scan(&currentStatus)
	if err != nil {
		return 0, err
	}
	return currentStatus, nil
}

// 用于定时任务：直接将最终统计出来的数字更新到 articles 对应的文章表里
func (r *articleLikeRepository) UpdateArticleLikeCountDirectly(ctx context.Context, articleID uint64, count uint32) error {
	return r.db.WithContext(ctx).
		Model(&model.Article{}). // 对应你的 model.Article 结构体
		Where("id = ?", articleID).
		Update("like_count", count).Error
}
