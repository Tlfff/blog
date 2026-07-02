package repository

import (
	"blog/internal/model"

	"gorm.io/gorm"
)

type ArticleRepository struct {
	db *gorm.DB
}

func NewArticleRepository(db *gorm.DB) *ArticleRepository {
	return &ArticleRepository{
		db: db,
	}
}

// 创建文章
// insert into articles
// (author_id, title, content, tags, status)
// values (?, ?, ?, ?, ?)
func (a *ArticleRepository) CreateArticle(article *model.Article) error {
	return a.db.Create(article).Error
}

// 更新文章（包括状态）
// update articles set title=?,content=?,status=?,tags=? where id=?
func (a *ArticleRepository) UpdateArticle(article *model.Article) error {
	return a.db.Model(&model.Article{}).
		Where("id=?", article.ID).
		Select("title", "content", "status", "tags").
		Updates(article).Error
}

// 逻辑删除文章
// update articles set status=? where id=?
func (a *ArticleRepository) DeleteArticle(articleId uint64) error {
	updateData := map[string]any{
		"status": model.Deleted,
	}
	return a.db.Model(&model.Article{}).
		Where("id=?", articleId).
		Updates(updateData).Error
}

// 硬删除文章
// delete from articles where id=?
func (a *ArticleRepository) ClearArticle(articleId uint64, userId uint64) error {
	return a.db.Table("articles").
		Where("id=?", articleId).
		Delete(nil).Error
}

// 根据id查找文章
// select id, author_id, title, content, tags, status, view_count, like_count, comment_count, created_time, update_time
// from articles where id=?
func (a *ArticleRepository) FindArticleByID(id uint64) (*model.Article, error) {
	var article model.Article
	err := a.db.Model(&model.Article{}).
		Select("id", "author_id", "title", "content", "tags", "status", "view_count", "like_count", "comment_count", "created_time", "updated_time").
		Where("id=?", id).
		Take(&article).Error
	return &article, err
}

// 列表查询
// select id, author_id, title, content, tags, status, view_count, like_count, comment_count, created_time, update_time
// from articles where author_id=? and status=?
func (a *ArticleRepository) GetListByStatus(AuthorID uint64, status int8) ([]*model.Article, error) {
	var list []*model.Article
	err := a.db.Model(&model.Article{}).
		Select("id", "author_id", "title", "content", "tags", "status", "view_count", "like_count", "comment_count", "created_time", "updated_time").
		Where("author_id=? AND status=?", AuthorID, status).
		Find(&list).Error
	return list, err
}
