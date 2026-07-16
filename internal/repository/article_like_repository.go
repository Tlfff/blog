package repository

import (
	"blog/internal/model"
	"context"
	"strings"

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
	GetLikesByID(ctx context.Context, articleID uint64) ([]*model.ArticleLike, error)
	CountValidByID(ctx context.Context, articleID uint64) (int64, error)
	UpdateArticleLikeCount(ctx context.Context, articleID uint64, count int64) error
	UpdateArticleLikeCountDirectly(ctx context.Context, articleID uint64, count uint32) error
	BatchInsertOrUpdateStatus(ctx context.Context, likes []*model.ArticleLike) error
	BatchUpdateArticleLikeCount(ctx context.Context, articles []*model.Article) error

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
func (r *articleLikeRepository) GetLikesByID(ctx context.Context, articleID uint64) ([]*model.ArticleLike, error) {
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

// 用于定时任务更新文章点赞总数（int64版本）
func (r *articleLikeRepository) UpdateArticleLikeCount(ctx context.Context, articleID uint64, count int64) error {
	return r.db.WithContext(ctx).
		Model(&model.Article{}).
		Where("id = ?", articleID).
		Update("like_count", count).Error
}

// 统计有效点赞总数 status=1
func (r *articleLikeRepository) CountValidByID(ctx context.Context, articleID uint64) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&model.ArticleLike{}).
		Where("article_id = ? AND status = ?", articleID, model.ArticleLiked).
		Count(&total).Error
	return total, err
}

// 评论插入，如果有则更新，批量
// insert into article_likes (user_id,article_id,status) values(?,?,?)
// on duplicate key update status=VALUES(status)
func (r *articleLikeRepository) BatchInsertOrUpdateStatus(ctx context.Context, likes []*model.ArticleLike) error {
	err := r.db.WithContext(ctx).
		Model(&model.ArticleLike{}).
		Select("user_id", "article_id", "status").
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "article_id"}}, // 唯一索引列
			DoUpdates: clause.AssignmentColumns([]string{"status"}),             //发生冲突（记录已存在）时，仅覆盖更新点赞状态字段
		}).
		CreateInBatches(likes, 100).Error // 100 条打包一组，多组自动在底层合并为批量 SQL

	return err
}

// 批量更新文章表的点赞字段
//
//	UPDATE articles
//	SET like_count = CASE WHEN id=? THEN ? WHEN id=? THEN ? ... END
//	WHERE id IN (?, ?, ...)
func (r *articleLikeRepository) BatchUpdateArticleLikeCount(ctx context.Context, articles []*model.Article) error {
	var caseSql string     // 拼接 case when 分支语句
	var ids []uint64       // 存储所有待更新文章id
	var args []interface{} // SQL参数数组，防止SQL注入

	// 循环构造每条记录的case匹配规则
	for _, article := range articles {
		caseSql += " WHEN id = ? THEN ? "
		ids = append(ids, article.ID)
		args = append(args, article.ID, article.LikeCount)
	}

	// 构造完整SQL
	sql := "UPDATE articles SET like_count = CASE " + caseSql + " END WHERE id IN (?" + strings.Repeat(",?", len(ids)-1) + ")"
	// 把id列表转为参数数组，拼接到末尾WHERE条件
	idArgs := make([]interface{}, len(ids))
	for i, v := range ids {
		idArgs[i] = v
	}
	args = append(args, idArgs...)

	// 执行原生SQL
	return r.db.WithContext(ctx).Exec(sql, args...).Error

}
