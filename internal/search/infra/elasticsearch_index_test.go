package infra

import (
	platformconfig "blog/internal/platform/config"
	platformelasticsearch "blog/internal/platform/elasticsearch"
	searchdomain "blog/internal/search/domain"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestElasticsearchIndexCreateAndSwitchAlias 验证显式 mapping 和原子别名切换请求。
func TestElasticsearchIndexCreateAndSwitchAlias(t *testing.T) {
	// 1. 启动记录索引创建和别名切换请求的测试服务
	var createBody map[string]any
	var aliasBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, _ := io.ReadAll(request.Body)
		switch request.URL.Path {
		case "/article_search_20260827093000":
			_ = json.Unmarshal(body, &createBody)
		case "/_aliases":
			_ = json.Unmarshal(body, &aliasBody)
		default:
			t.Fatalf("收到未预期请求: %s %s", request.Method, request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"acknowledged":true}`))
	}))
	defer server.Close()
	index, closeClient := newTestIndex(t, server.URL)
	defer closeClient()

	// 2. 创建物理索引并切换稳定别名
	ctx := context.Background()
	if err := index.CreateIndex(ctx, "article_search_20260827093000"); err != nil {
		t.Fatalf("创建测试索引失败: %v", err)
	}
	if err := index.SwitchAlias(ctx, "article_search", "article_search_20260827093000"); err != nil {
		t.Fatalf("切换测试别名失败: %v", err)
	}
	mappings := createBody["mappings"].(map[string]any)
	if mappings["dynamic"] != "strict" {
		t.Fatalf("索引未禁用动态字段扩张: %+v", mappings)
	}
	settingsBytes, _ := json.Marshal(createBody["settings"])
	for _, expected := range []string{
		"keep_joined_full_pinyin", "article_pinyin_initial_tokenizer", "keep_separate_first_letter", "limit_first_letter_length",
	} {
		if !strings.Contains(string(settingsBytes), expected) {
			t.Fatalf("拼音配置缺少 %s: %s", expected, settingsBytes)
		}
	}
	mappingBytes, _ := json.Marshal(createBody["mappings"])
	if !strings.Contains(string(mappingBytes), "pinyin_initial") {
		t.Fatalf("标题首字母 multi-field 缺失: %s", mappingBytes)
	}
	if len(aliasBody["actions"].([]any)) != 2 {
		t.Fatalf("别名切换未包含移除和新增动作: %+v", aliasBody)
	}
}

// TestElasticsearchIndexSearch 验证固定权重、高亮和响应解析。
func TestElasticsearchIndexSearch(t *testing.T) {
	// 1. 返回包含标题高亮、正文摘要和精确总数的搜索响应
	var searchBody string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, _ := io.ReadAll(request.Body)
		searchBody = string(body)
		if request.URL.Path != "/article_search/_search" || request.URL.RawQuery != "" {
			t.Fatalf("搜索请求不符合预期: %s?%s", request.URL.Path, request.URL.RawQuery)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"hits":{"total":{"value":12},"hits":[{"_id":"7","_source":{"article_id":7,"title":"Canal 搜索","content":"纯文本正文","tags":["Go"],"updated_time":"2026-08-27T00:00:00Z"},"highlight":{"title":["<em>Canal</em> 搜索"],"content":["纯文本<em>正文</em>"]}}]}}`))
	}))
	defer server.Close()
	index, closeClient := newTestIndex(t, server.URL)
	defer closeClient()

	// 2. 查询第二页并核对权重和高亮转换
	result, err := index.Search(context.Background(), searchdomain.SearchQuery{Keyword: "canal", Page: 2, PageSize: 10})
	if err != nil {
		t.Fatalf("查询测试索引失败: %v", err)
	}
	for _, expected := range []string{
		`"boost":5`, `"boost":3`, `"boost":2`, `"boost":1`,
		`"title.pinyin"`, `"title.pinyin_initial"`, `"track_total_hits":true`,
		`"from":10`, `"size":10`,
	} {
		if !strings.Contains(searchBody, expected) {
			t.Fatalf("搜索请求缺少固定查询配置 %s: %s", expected, searchBody)
		}
	}
	if result.Total != 12 || len(result.Hits) != 1 || result.Hits[0].TitleHighlight == "" || result.Hits[0].Summary == "" {
		t.Fatalf("搜索响应解析不符合预期: %+v", result)
	}
}

// TestElasticsearchIndexBulkPartialFailure 验证 Bulk 部分失败和幂等删除。
func TestElasticsearchIndexBulkPartialFailure(t *testing.T) {
	// 1. 按调用次数返回写入失败和删除 404 响应
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount++
		writer.Header().Set("Content-Type", "application/json")
		if requestCount == 1 {
			_, _ = writer.Write([]byte(`{"errors":true,"items":[{"index":{"_id":"1","status":400,"error":{"type":"mapper_parsing_exception"}}}]}`))
			return
		}
		_, _ = writer.Write([]byte(`{"errors":true,"items":[{"delete":{"_id":"9","status":404,"error":{"type":"document_missing_exception"}}}]}`))
	}))
	defer server.Close()
	index, closeClient := newTestIndex(t, server.URL)
	defer closeClient()

	// 2. 部分写入失败返回错误，删除不存在文档保持成功
	err := index.BulkUpsert(context.Background(), "", []searchdomain.ArticleDocument{{ArticleID: 1, Title: "标题"}})
	if err == nil || !strings.Contains(err.Error(), "文档 1") {
		t.Fatalf("Bulk 部分失败错误不符合预期: %v", err)
	}
	if err := index.DeleteDocuments(context.Background(), "", []uint64{9}); err != nil {
		t.Fatalf("删除不存在文档未保持幂等: %v", err)
	}
}

// TestElasticsearchIndexUnavailable 验证网络错误保留底层错误链。
func TestElasticsearchIndexUnavailable(t *testing.T) {
	// 1. 创建后立即关闭服务以模拟连接失败
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	index, closeClient := newTestIndex(t, server.URL)
	server.Close()
	defer closeClient()

	// 2. 搜索失败返回可识别的网络错误
	_, err := index.Search(context.Background(), searchdomain.SearchQuery{Keyword: "go", Page: 1, PageSize: 10})
	if err == nil {
		t.Fatal("Elasticsearch 不可用时未返回错误")
	}
	var networkErr interface{ Timeout() bool }
	if !errors.As(err, &networkErr) && !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("搜索网络错误链不符合预期: %v", err)
	}
}

// newTestIndex 创建指向测试服务的 Elasticsearch Adapter。
func newTestIndex(t *testing.T, address string) (*ElasticsearchIndex, func()) {
	// 1. 创建不执行在线探测的平台客户端
	t.Helper()
	client, err := platformelasticsearch.NewClient(platformconfig.Elasticsearch{Addr: address, IndexAlias: "article_search", RequestTimeoutMS: 500})
	if err != nil {
		t.Fatalf("创建测试 Elasticsearch 客户端失败: %v", err)
	}
	return NewElasticsearchIndex(client, "article_search"), func() { _ = client.Close() }
}
