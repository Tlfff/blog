package app

import (
	searchdomain "blog/internal/search/domain"
	"context"
	"errors"
	"testing"
)

// fakeSearcher 返回预设搜索结果或错误。
type fakeSearcher struct {
	query  searchdomain.SearchQuery   // 收到的搜索条件
	result *searchdomain.SearchResult // 预设搜索结果
	err    error                      // 预设搜索错误
}

// Search 记录查询并返回预设结果。
func (f *fakeSearcher) Search(_ context.Context, query searchdomain.SearchQuery) (*searchdomain.SearchResult, error) {
	// 1. 保存查询条件并返回预设值
	f.query = query
	return f.result, f.err
}

// TestServiceSearchArticles 验证查询校验和响应转换。
func TestServiceSearchArticles(t *testing.T) {
	// 1. 准备包含高亮的搜索结果
	searcher := &fakeSearcher{result: &searchdomain.SearchResult{
		Hits:  []searchdomain.SearchHit{{ArticleID: 1, Title: "Canal 搜索", TitleHighlight: "<em>Canal</em> 搜索", Tags: []string{"Go"}}},
		Total: 1,
	}}
	service := NewService(searcher)

	// 2. 查询时裁剪关键词并转换响应
	response, err := service.SearchArticles(context.Background(), " Canal ", 1, 10)
	if err != nil {
		t.Fatalf("搜索文章失败: %v", err)
	}
	if searcher.query.Keyword != "Canal" || response.Total != 1 || response.List[0].TitleHighlight == "" {
		t.Fatalf("搜索结果不符合预期: query=%+v response=%+v", searcher.query, response)
	}
}

// TestServiceSearchArticlesValidation 验证空关键词和分页边界。
func TestServiceSearchArticlesValidation(t *testing.T) {
	// 1. 分别验证关键词、页码和每页数量错误
	service := NewService(&fakeSearcher{})
	tests := []struct {
		name     string // 测试场景名称
		keyword  string // 搜索关键词
		page     uint64 // 页码
		pageSize uint64 // 每页数量
		wantErr  error  // 预期错误
	}{
		{name: "空关键词", keyword: " ", page: 1, pageSize: 10, wantErr: searchdomain.ErrKeywordEmpty},
		{name: "页码为零", keyword: "go", page: 0, pageSize: 10, wantErr: searchdomain.ErrPageInvalid},
		{name: "每页过小", keyword: "go", page: 1, pageSize: 9, wantErr: searchdomain.ErrPageSizeInvalid},
		{name: "每页过大", keyword: "go", page: 1, pageSize: 21, wantErr: searchdomain.ErrPageSizeInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := service.SearchArticles(context.Background(), test.keyword, test.page, test.pageSize)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("错误不符合预期: got=%v want=%v", err, test.wantErr)
			}
		})
	}
}

// TestServiceSearchUnavailablePreservesCause 验证搜索错误隔离并保留错误链。
func TestServiceSearchUnavailablePreservesCause(t *testing.T) {
	// 1. 准备底层搜索错误
	searchErr := errors.New("连接 ES 失败")
	service := NewService(&fakeSearcher{err: searchErr})

	// 2. 返回统一不可用错误并保留底层根因
	_, err := service.SearchArticles(context.Background(), "go", 1, 10)
	if !errors.Is(err, searchdomain.ErrSearchUnavailable) || !errors.Is(err, searchErr) {
		t.Fatalf("搜索错误链不符合预期: %v", err)
	}
}

// TestServicePreservesOptionalHighlights 验证拼音或标签单独命中允许空高亮和摘要。
func TestServicePreservesOptionalHighlights(t *testing.T) {
	// 1. 准备只有原始标题和标签的搜索命中
	service := NewService(&fakeSearcher{result: &searchdomain.SearchResult{
		Hits:  []searchdomain.SearchHit{{ArticleID: 2, Title: "深入理解", Tags: []string{"Go"}}},
		Total: 1,
	}})

	// 2. 响应保持空标题高亮和空正文摘要
	response, err := service.SearchArticles(context.Background(), "shenrulijie", 1, 10)
	if err != nil {
		t.Fatalf("查询拼音命中结果失败: %v", err)
	}
	if response.List[0].TitleHighlight != "" || response.List[0].Summary != "" {
		t.Fatalf("可选高亮字段未保持为空: %+v", response.List[0])
	}
}

// TestServicePreservesInitialPinyinEmptyHighlight 验证首字母简写命中不伪造中文标题高亮。
func TestServicePreservesInitialPinyinEmptyHighlight(t *testing.T) {
	// 1. 准备只有原始标题的首字母简写搜索命中
	searcher := &fakeSearcher{result: &searchdomain.SearchResult{
		Hits:  []searchdomain.SearchHit{{ArticleID: 3, Title: "深入理解"}},
		Total: 1,
	}}
	service := NewService(searcher)

	// 2. 查询 srlj 后保持原始标题，标题高亮和摘要为空
	response, err := service.SearchArticles(context.Background(), "srlj", 1, 10)
	if err != nil {
		t.Fatalf("查询拼音首字母命中结果失败: %v", err)
	}
	if searcher.query.Keyword != "srlj" || response.List[0].Title != "深入理解" {
		t.Fatalf("拼音首字母查询或标题不符合预期: query=%+v item=%+v", searcher.query, response.List[0])
	}
	if response.List[0].TitleHighlight != "" || response.List[0].Summary != "" {
		t.Fatalf("首字母命中产生了不应存在的高亮: %+v", response.List[0])
	}
}
