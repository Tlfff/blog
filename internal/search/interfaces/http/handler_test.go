package http

import (
	"blog/internal/platform/interfaces/http/middleware"
	searchdto "blog/internal/search/app/dto"
	searchdomain "blog/internal/search/domain"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// fakeSearchUsecase 返回预设文章搜索响应或错误。
type fakeSearchUsecase struct {
	keyword  string                           // 收到的搜索关键词
	page     uint64                           // 收到的页码
	pageSize uint64                           // 收到的每页数量
	response *searchdto.ArticleSearchResponse // 预设搜索响应
	err      error                            // 预设搜索错误
}

// SearchArticles 记录搜索条件并返回预设结果。
func (f *fakeSearchUsecase) SearchArticles(_ context.Context, keyword string, page, pageSize uint64) (*searchdto.ArticleSearchResponse, error) {
	// 1. 保存查询条件并返回测试结果
	f.keyword = keyword
	f.page = page
	f.pageSize = pageSize
	return f.response, f.err
}

// TestHandlerSearchArticles 验证前台搜索成功响应契约。
func TestHandlerSearchArticles(t *testing.T) {
	// 1. 准备包含标题高亮、正文摘要和标签的响应
	gin.SetMode(gin.TestMode)
	usecase := &fakeSearchUsecase{response: &searchdto.ArticleSearchResponse{
		List: []searchdto.ArticleSearchItem{{
			ID: 1, Title: "Canal 搜索", TitleHighlight: "<em>Canal</em> 搜索", Summary: "正文摘要", Tags: []string{"Go"},
		}},
		Total: 1, Page: 1, PageSize: 10,
	}}
	router := gin.New()
	router.Use(middleware.GlobalErrorMiddleware())
	router.GET("/article/search", NewHandler(usecase).SearchArticles)

	// 2. 请求搜索接口并核对统一响应字段
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/article/search?keyword=canal&page=1&page_size=10", nil)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("搜索接口 HTTP 状态码错误: %d", recorder.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("搜索响应 JSON 非法: %v", err)
	}
	if body["success"] != true || usecase.keyword != "canal" || usecase.page != 1 || usecase.pageSize != 10 {
		t.Fatalf("搜索成功响应不符合预期: body=%s usecase=%+v", recorder.Body.String(), usecase)
	}
}

// TestHandlerSearchArticlesValidation 验证空关键词和分页参数错误。
func TestHandlerSearchArticlesValidation(t *testing.T) {
	// 1. 定义缺失关键词、非法页码和非法每页数量请求
	gin.SetMode(gin.TestMode)
	tests := []string{
		"/article/search?page=1&page_size=10",
		"/article/search?keyword=go&page=0&page_size=10",
		"/article/search?keyword=go&page=1&page_size=9",
	}
	for _, target := range tests {
		router := gin.New()
		router.Use(middleware.GlobalErrorMiddleware())
		router.GET("/article/search", NewHandler(&fakeSearchUsecase{}).SearchArticles)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
		if !strings.Contains(recorder.Body.String(), `"code":1001`) {
			t.Fatalf("非法搜索参数响应不符合预期: %s", recorder.Body.String())
		}
	}
}

// TestHandlerSearchUnavailable 验证搜索故障返回独立业务错误且不降级。
func TestHandlerSearchUnavailable(t *testing.T) {
	// 1. 准备搜索不可用错误
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.GlobalErrorMiddleware())
	router.GET("/article/search", NewHandler(&fakeSearchUsecase{err: searchdomain.ErrSearchUnavailable}).SearchArticles)

	// 2. 接口返回搜索不可用业务码
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/article/search?keyword=go&page=1&page_size=10", nil))
	if !strings.Contains(recorder.Body.String(), `"code":1700`) {
		t.Fatalf("搜索不可用响应不符合预期: %s", recorder.Body.String())
	}
}
