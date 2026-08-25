package infra

import (
	domaincontent "blog/internal/article/domain"
	"blog/internal/article/infra/model"
	"context"
	"errors"

	"gorm.io/gorm"
)

// articleRepository 是 Content 文章仓储的 GORM 实现。
type articleRepository struct {
	db *gorm.DB // GORM 数据库连接
}

// NewArticleRepository 返回直接持有 GORM 的 Content 文章 Repository 实现。
func NewArticleRepository(db *gorm.DB) domaincontent.ArticleRepository {
	return &articleRepository{db: db}
}

// Create 创建文章，并把数据库生成的主键与时间回填到领域对象。
func (r *articleRepository) Create(ctx context.Context, article *domaincontent.Article) error {
	// 1. 领域对象转换为数据库模型
	m := toModelArticle(article)
	// 2. 写入数据库
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	// 3. 回填自增ID与时间字段
	article.ID = m.ID
	article.CreatedTime = m.CreatedTime
	article.UpdatedTime = m.UpdatedTime
	return nil
}

// FindByID 按文章 ID 查询文章，查不到时返回文章不存在的领域错误。
func (r *articleRepository) FindByID(ctx context.Context, id uint64) (*domaincontent.Article, error) {
	var m model.Article
	err := r.db.WithContext(ctx).Model(&model.Article{}).
		Select("id", "author_id", "title", "content", "tags", "status", "view_count", "like_count", "comment_count", "created_time", "updated_time").
		Where("id=?", id).
		Take(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domaincontent.ErrArticleNotFound
		}
		return nil, err
	}
	return toDomainArticle(&m), nil
}

// FindWithAuthorByID 使用调用方拥有的只读 JOIN 查询文章详情和作者展示字段。
func (r *articleRepository) FindWithAuthorByID(ctx context.Context, id uint64) (*domaincontent.ArticleWithAuthor, error) {
	// 1. 执行字段最小化的只读 JOIN，不复用 User Repository 或持久化模型
	var row articleWithAuthorRow
	err := r.db.WithContext(ctx).Table("articles a").
		Select(`a.id, a.author_id, a.title, a.content, a.tags, a.status, a.view_count, a.like_count, a.comment_count, a.created_time, a.updated_time,
			u.nickname AS nickname, u.avatar AS avatar, u.last_login_ip AS last_login_ip`).
		Joins("LEFT JOIN users u ON a.author_id = u.id").
		Where("a.id = ?", id).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domaincontent.ErrArticleNotFound
		}
		return nil, err
	}

	// 2. 转换为 Article 拥有的详情 Read Model
	article := toDomainArticle(&row.Article)
	return &domaincontent.ArticleWithAuthor{
		Article: *article, Nickname: row.Nickname, Avatar: row.Avatar, LastLoginIP: row.LastLoginIP,
	}, nil
}

// articleWithAuthorRow 表示文章详情只读 JOIN 的行映射。
type articleWithAuthorRow struct {
	model.Article        // 文章表字段
	Nickname      string `gorm:"column:nickname"`      // 作者昵称
	Avatar        string `gorm:"column:avatar"`        // 作者头像
	LastLoginIP   string `gorm:"column:last_login_ip"` // 作者最后登录 IP
}

// Update 更新文章，仅更新标题、正文、状态与标签字段。
func (r *articleRepository) Update(ctx context.Context, article *domaincontent.Article) error {
	return r.db.WithContext(ctx).Model(&model.Article{}).
		Where("id=?", article.ID).
		Select("title", "content", "status", "tags").
		Updates(toModelArticle(article)).Error
}

// SoftDelete 软删除文章，只把状态改为已删除，数据仍保留在库中。
func (r *articleRepository) SoftDelete(ctx context.Context, articleID uint64) error {
	return r.db.WithContext(ctx).Model(&model.Article{}).
		Where("id=?", articleID).
		Updates(map[string]any{"status": model.Deleted}).Error
}

// Clear 物理删除文章，彻底清除数据库记录。
func (r *articleRepository) Clear(ctx context.Context, articleID uint64) error {
	return r.db.WithContext(ctx).Table("articles").
		Where("id=?", articleID).
		Delete(nil).Error
}

// ListWithCursor 使用游标分页查询文章列表。
//
// 参数说明：
//   - ctx：请求上下文，用于传递链路信息和控制超时。
//   - lastID：游标文章唯一标识。
//   - pageSize：每页数量。
//   - isDesc：是否按文章唯一标识倒序排列。
//   - status：文章状态过滤值。
func (r *articleRepository) ListWithCursor(ctx context.Context, lastID uint64, pageSize int, isDesc bool, status int8) ([]*domaincontent.Article, error) {
	// 1. 限定查询字段
	tx := r.db.WithContext(ctx).Model(&model.Article{}).
		Select("id", "author_id", "title", "content", "tags", "status", "view_count", "like_count", "comment_count", "created_time", "updated_time")
	// 2. 追加状态过滤条件
	tx = applyStatusCondition(tx, status)
	// 3. 按排序方向决定游标比较方式
	if isDesc {
		tx = tx.Where("id < ?", lastID).Order("id DESC")
	} else {
		tx = tx.Where("id > ?", lastID).Order("id ASC")
	}
	// 4. 按页大小查询并转换为领域对象
	var models []*model.Article
	if err := tx.Limit(pageSize).Find(&models).Error; err != nil {
		return nil, err
	}
	return toDomainArticles(models), nil
}

// ListWithOffset 使用 Offset 分页查询文章列表。
//
// 参数说明：
//   - ctx：请求上下文，用于传递链路信息和控制超时。
//   - page：当前页码。
//   - pageSize：每页数量。
//   - isDesc：是否按文章唯一标识倒序排列。
//   - status：文章状态过滤值。
func (r *articleRepository) ListWithOffset(ctx context.Context, page, pageSize int, isDesc bool, status int8) ([]*domaincontent.Article, error) {
	// 1. 限定查询字段
	tx := r.db.WithContext(ctx).Model(&model.Article{}).
		Select("id", "author_id", "title", "content", "tags", "status", "view_count", "like_count", "comment_count", "created_time", "updated_time")
	// 2. 追加状态过滤条件
	tx = applyStatusCondition(tx, status)
	// 3. 按 isDesc 决定排序方向
	if isDesc {
		tx = tx.Order("id DESC")
	} else {
		tx = tx.Order("id ASC")
	}
	// 4. 按页码换算 offset 后查询并转换为领域对象
	var models []*model.Article
	if err := tx.Limit(pageSize).Offset((page - 1) * pageSize).Find(&models).Error; err != nil {
		return nil, err
	}
	return toDomainArticles(models), nil
}

// CountByStatus 按状态统计文章总数。
func (r *articleRepository) CountByStatus(ctx context.Context, status int8) (int64, error) {
	var count int64
	tx := r.db.WithContext(ctx).Model(&model.Article{})
	tx = applyStatusCondition(tx, status)
	if err := tx.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// applyStatusCondition 按状态过滤值追加查询条件。
func applyStatusCondition(tx *gorm.DB, status int8) *gorm.DB {
	switch status {
	case domaincontent.StatusAll:
		return tx
	case domaincontent.StatusAllExceptDeleted:
		return tx.Where("status IN ?", []int8{domaincontent.StatusDraft.Int8(), domaincontent.StatusPublished.Int8()})
	default:
		return tx.Where("status = ?", status)
	}
}

// toDomainArticle 把数据库模型转换为 Article 领域文章对象。
func toDomainArticle(m *model.Article) *domaincontent.Article {
	return &domaincontent.Article{
		ID:           m.ID,
		AuthorID:     m.AuthorID,
		Title:        m.Title,
		Content:      m.Content,
		Tags:         m.Tags,
		Status:       domaincontent.ArticleStatus(m.Status),
		ViewCount:    m.ViewCount,
		LikeCount:    m.LikeCount,
		CommentCount: m.CommentCount,
		CreatedTime:  m.CreatedTime,
		UpdatedTime:  m.UpdatedTime,
	}
}

// toModelArticle 把 Article 领域文章对象转换为数据库模型。
func toModelArticle(a *domaincontent.Article) *model.Article {
	return &model.Article{
		ID:           a.ID,
		AuthorID:     a.AuthorID,
		Title:        a.Title,
		Content:      a.Content,
		Tags:         a.Tags,
		Status:       a.Status.Int8(),
		ViewCount:    a.ViewCount,
		LikeCount:    a.LikeCount,
		CommentCount: a.CommentCount,
		CreatedTime:  a.CreatedTime,
		UpdatedTime:  a.UpdatedTime,
	}
}

// toDomainArticles 批量把数据库模型转换为领域文章对象。
func toDomainArticles(models []*model.Article) []*domaincontent.Article {
	articles := make([]*domaincontent.Article, 0, len(models))
	for _, m := range models {
		articles = append(articles, toDomainArticle(m))
	}
	return articles
}
