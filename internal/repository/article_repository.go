package repository

import (
	"blog/internal/model"

	"context"

	"gorm.io/gorm"
)

type ArticleWithUser struct {
	model.Article
	Nickname    string `gorm:"column:nickname"`
	Avatar      string `gorm:"column:avatar"`
	LastLoginIp string `gorm:"column:last_login_ip"`
}

type ArticleRepository struct {
	db *gorm.DB
}

func NewArticleRepository(db *gorm.DB) *ArticleRepository {
	return &ArticleRepository{
		db: db,
	}
}

// 连表查单条文章详情
// SELECT a.id, a.author_id, a.title, a.content, a.tags, a.status, a.view_count, a.like_count, a.comment_count, a.created_time, a.updated_time,
// u.nickname, u.avatar, u.last_login_ip
// FROM articles a
// LEFT JOIN users u ON a.author_id = u.id
// WHERE a.id = ?
// LIMIT 1
func (a *ArticleRepository) FindArticleAndUserInfoByID(ctx context.Context, id uint64) (*ArticleWithUser, error) {
	var result ArticleWithUser
	err := a.db.WithContext(ctx).Table("articles a").
		Select(`a.id, a.author_id, a.title, a.content, a.tags, a.status, a.view_count, a.like_count, a.comment_count, a.created_time, a.updated_time, u.nickname, u.avatar, u.last_login_ip`).
		Joins("LEFT JOIN users u ON a.author_id = u.id").
		Where("a.id = ?", id).
		Take(&result).Error
	return &result, err
}

// 创建文章
// insert into articles
// (author_id, title, content, tags, status)
// values (?, ?, ?, ?, ?)
func (a *ArticleRepository) CreateArticle(ctx context.Context, article *model.Article) error {
	return a.db.Create(article).Error
}

// 更新文章（包括状态）
// update articles set title=?,content=?,status=?,tags=? where id=?
func (a *ArticleRepository) UpdateArticle(ctx context.Context, article *model.Article) error {
	return a.db.WithContext(ctx).Model(&model.Article{}).
		Where("id=?", article.ID).
		Select("title", "content", "status", "tags").
		Updates(article).Error
}

// 逻辑删除文章
// update articles set status=? where id=?
func (a *ArticleRepository) DeleteArticle(ctx context.Context, articleId uint64) error {
	updateData := map[string]any{
		"status": model.Deleted,
	}
	return a.db.WithContext(ctx).Model(&model.Article{}).
		Where("id=?", articleId).
		Updates(updateData).Error
}

// 硬删除文章
// delete from articles where id=?
func (a *ArticleRepository) ClearArticle(ctx context.Context, articleId uint64, userId uint64) error {
	return a.db.WithContext(ctx).Table("articles").
		Where("id=?", articleId).
		Delete(nil).Error
}

// 根据id查找文章
// select id, author_id, title, content, tags, status, view_count, like_count, comment_count, created_time, update_time
// from articles where id=?
func (a *ArticleRepository) FindArticleByID(ctx context.Context, id uint64) (*model.Article, error) {
	var article model.Article
	err := a.db.WithContext(ctx).Model(&model.Article{}).
		Select("id", "author_id", "title", "content", "tags", "status", "view_count", "like_count", "comment_count", "created_time", "updated_time").
		Where("id=?", id).
		Take(&article).Error
	return &article, err
}

// 列表查询
// select
//
//	id, author_id, title, content, tags, status, view_count, like_count, comment_count, created_time, update_time
//
// from articles
// where author_id=? and status=?
func (a *ArticleRepository) GetListByStatus(ctx context.Context, AuthorID uint64, status int8) ([]*model.Article, error) {
	var list []*model.Article
	err := a.db.WithContext(ctx).Model(&model.Article{}).
		Select("id", "author_id", "title", "content", "tags", "status", "view_count", "like_count", "comment_count", "created_time", "updated_time").
		Where("author_id=? AND status=?", AuthorID, status).
		Find(&list).Error
	return list, err
}

// 游标分页查询
// select
//
//	id, author_id, title, content, tags, status, view_count, like_count, comment_count, created_time, update_time
//
// from articles
// where  status=?
// -- 游标条件：isDesc=true 拼接 AND articles.id < ? ORDER BY articles.id DESC
// -- 游标条件：isDesc=false 拼接 AND articles.id > ? ORDER BY articles.id ASC
// limit ?
func (a *ArticleRepository) GetListWithCursor(ctx context.Context, lastID uint64, pageSize int, isDesc bool, status int8) ([]*model.Article, error) {
	var list []*model.Article
	tx := a.db.WithContext(ctx).Model(&model.Article{}).
		Select("id", "author_id", "title", "content", "tags", "status", "view_count", "like_count", "comment_count", "created_time", "updated_time")
	tx = applyStatusCondition(tx, status)
	if isDesc {
		tx = tx.Where("id < ?", lastID).Order("id DESC")
	} else {
		tx = tx.Where("id > ?", lastID).Order("id ASC")
	}
	err := tx.Limit(pageSize).Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// 传统分页查询
// select
//
//	id, author_id, title, content, tags, status, view_count, like_count, comment_count, created_time, update_time
//
// from articles
// where status=?
// -- isDesc=true: ORDER BY c.id DESC
// -- isDesc=false: ORDER BY c.id ASC
// LIMIT ? OFFSET ?;
func (a *ArticleRepository) GetListWithOffset(ctx context.Context, page int, pageSize int, isDesc bool, status int8) ([]*model.Article, error) {
	var list []*model.Article
	tx := a.db.WithContext(ctx).Model(&model.Article{}).
		Select("id", "author_id", "title", "content", "tags", "status", "view_count", "like_count", "comment_count", "created_time", "updated_time")
	tx = applyStatusCondition(tx, status)
	if isDesc {
		tx = tx.Order("id DESC")
	} else {
		tx = tx.Order("id ASC")
	}
	offset := (page - 1) * pageSize
	err := tx.Limit(pageSize).Offset(offset).Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// 更新文章评论数
// 传入 delta 改变文章评论数（delta 可以是正数如 1，也可以是负数如 -5）
//
// 使用传入的 tx 确保操作在外部的事务生命周期内
func (a *ArticleRepository) UpdateCommentCountDelta(ctx context.Context, tx *gorm.DB, articleID uint64, delta int64) error {
	return tx.WithContext(ctx).Model(&model.Article{}).
		Where("id = ?", articleID).
		Update("comment_count", gorm.Expr("comment_count + ?", delta)).Error
}

// 查找文章数量
// SELECT COUNT(*) FROM articles WHERE status = ?
func (a *ArticleRepository) GetArticleCountByStatus(ctx context.Context, status int8) (int64, error) {
	var count int64
	tx := a.db.WithContext(ctx).Model(&model.Article{})
	tx = applyStatusCondition(tx, status)
	err := tx.Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

// 转换传入的文章状态
func applyStatusCondition(tx *gorm.DB, status int8) *gorm.DB {
	switch status {
	case model.All: // 全部（含删除）-> 此时不需要拼任何 status 条件，放行全部数据
		return tx
	case model.AllExceptDeleted: // 全部（不含删除）
		return tx.Where("status IN ?", []int8{model.Draft, model.Published})
	default: // 具体某个状态（如 1, 2, 3）
		return tx.Where("status = ?", status)
	}
}
