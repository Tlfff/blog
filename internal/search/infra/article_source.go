package infra

import (
	searchdomain "blog/internal/search/domain"
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// articleSource 使用 Search 自有只读模型查询已发表文章。
type articleSource struct {
	db *gorm.DB // 默认 GORM 数据库连接
}

// articleSourceRow 表示搜索全量重建所需的最小文章行。
type articleSourceRow struct {
	ID          uint64                           `gorm:"column:id"`           // 文章唯一标识
	Title       string                           `gorm:"column:title"`        // 文章原始标题
	Content     string                           `gorm:"column:content"`      // Markdown 正文
	Tags        string                           `gorm:"column:tags"`         // 英文逗号分隔的标签字符串
	Status      searchdomain.ArticleSourceStatus `gorm:"column:status"`       // 文章状态：1-已删除；2-草稿；3-已发表
	UpdatedTime time.Time                        `gorm:"column:updated_time"` // 文章最后更新时间
}

// NewArticleSource 创建 Search 自有的 MySQL 只读文章数据源。
func NewArticleSource(db *gorm.DB) searchdomain.ArticleSource {
	// 1. 保存调用方拥有的只读数据库连接
	return &articleSource{db: db}
}

// ListPublishedAfter 按文章 ID 游标分批读取已发表文章。
func (s *articleSource) ListPublishedAfter(ctx context.Context, lastID uint64, limit int) ([]searchdomain.SourceArticle, error) {
	// 1. 校验分页大小并查询已发表文章最小字段
	if limit <= 0 {
		return nil, fmt.Errorf("文章搜索重建批次大小必须大于 0")
	}
	var rows []articleSourceRow
	err := s.db.WithContext(ctx).Table("articles").
		Select("id", "title", "content", "tags", "status", "updated_time").
		Where("status = ? AND id > ?", searchdomain.ArticleStatusPublished, lastID).
		Order("id ASC").Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("读取已发表文章搜索数据失败: %w", err)
	}

	// 2. 转换为 Search 自有来源模型
	articles := make([]searchdomain.SourceArticle, 0, len(rows))
	for _, row := range rows {
		articles = append(articles, searchdomain.SourceArticle{
			ID: row.ID, Title: row.Title, Content: row.Content, Tags: row.Tags,
			Status: row.Status, UpdatedTime: row.UpdatedTime,
		})
	}
	return articles, nil
}

// CountPublished 统计当前已发表文章数量。
func (s *articleSource) CountPublished(ctx context.Context) (uint64, error) {
	// 1. 只统计当前已发表文章
	var count int64
	if err := s.db.WithContext(ctx).Table("articles").Where("status = ?", searchdomain.ArticleStatusPublished).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计已发表文章数量失败: %w", err)
	}
	return uint64(count), nil
}
