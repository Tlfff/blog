package infra

import (
	domaincommunity "blog/internal/article/domain"
	"blog/internal/article/infra/model"
	"context"

	"gorm.io/gorm"
)

// articleQueryAdapter 是 Article 上下文排行查询 Port 的 GORM 实现。
type articleQueryAdapter struct {
	db *gorm.DB // GORM 数据库连接
}

// NewRankingQuery 创建 Article 排行查询适配器。
func NewRankingQuery(db *gorm.DB) domaincommunity.RankingQuery {
	return &articleQueryAdapter{db: db}
}

// GetHotListByIDs 按 ID 批量查询文章统计数据。
func (a *articleQueryAdapter) GetHotListByIDs(ctx context.Context, ids []uint64) ([]*domaincommunity.ArticleInfo, error) {
	var models []*model.Article
	err := a.db.WithContext(ctx).Model(&model.Article{}).
		Select("id,title, view_count, comment_count, like_count").
		Where("id IN ?", ids).
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	return toArticleInfos(models), nil
}

// GetTopHotArticles 查询已发表的热门文章。
func (a *articleQueryAdapter) GetTopHotArticles(ctx context.Context, limit int) ([]*domaincommunity.ArticleInfo, error) {
	var models []*model.Article
	err := a.db.WithContext(ctx).Model(&model.Article{}).
		Select("id, view_count, comment_count, like_count").
		Where("status = ?", model.Published).
		Order("(view_count + like_count + comment_count) DESC").
		Limit(limit).
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	return toArticleInfos(models), nil
}

// 把文章数据模型转换为 Community 只读查询模型
func toArticleInfo(m *model.Article) *domaincommunity.ArticleInfo {
	return &domaincommunity.ArticleInfo{
		ID:           m.ID,
		AuthorID:     m.AuthorID,
		Title:        m.Title,
		ViewCount:    m.ViewCount,
		LikeCount:    m.LikeCount,
		CommentCount: m.CommentCount,
	}
}

// 批量把文章数据模型转换为 Community 只读查询模型
func toArticleInfos(articles []*model.Article) []*domaincommunity.ArticleInfo {
	list := make([]*domaincommunity.ArticleInfo, 0, len(articles))
	for _, article := range articles {
		list = append(list, toArticleInfo(article))
	}
	return list
}
