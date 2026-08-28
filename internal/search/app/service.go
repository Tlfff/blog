package app

import (
	searchdto "blog/internal/search/app/dto"
	searchdomain "blog/internal/search/domain"
	"context"
	"fmt"
)

// Service 提供前台文章搜索应用用例。
type Service struct {
	searcher searchdomain.IndexSearcher // Elasticsearch 搜索查询 Port
}

// NewService 创建文章搜索应用服务。
func NewService(searcher searchdomain.IndexSearcher) *Service {
	// 1. 保存搜索查询依赖
	return &Service{searcher: searcher}
}

// SearchArticles 校验搜索条件并返回前台分页结果。
func (s *Service) SearchArticles(ctx context.Context, keyword string, page, pageSize uint64) (*searchdto.ArticleSearchResponse, error) {

	// 1. 查询搜索引擎并隔离内部错误细节
	result, err := s.searcher.Search(ctx, searchdomain.SearchQuery{
		Keyword:  keyword,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", searchdomain.ErrSearchUnavailable, err)
	}

	// 2. 转换为稳定 HTTP 响应 DTO
	items := make([]searchdto.ArticleSearchItem, 0, len(result.Hits))
	for _, hit := range result.Hits {
		items = append(items, searchdto.ArticleSearchItem{
			ID:             hit.ArticleID,
			Title:          hit.Title,
			TitleHighlight: hit.TitleHighlight,
			Summary:        hit.Summary,
			Tags:           hit.Tags,
		})
	}
	return &searchdto.ArticleSearchResponse{
		List: items, Total: result.Total, Page: page, PageSize: pageSize,
	}, nil
}
