package infra

import (
	platformelasticsearch "blog/internal/platform/elasticsearch"
	searchdomain "blog/internal/search/domain"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8/esapi"
)

// ElasticsearchIndex 实现文章搜索查询、文档写入和索引管理能力。
type ElasticsearchIndex struct {
	client *platformelasticsearch.Client // 平台层 Elasticsearch 客户端
	alias  string                        // 文章搜索稳定索引别名
}

// NewElasticsearchIndex 创建文章搜索 Elasticsearch Adapter。
func NewElasticsearchIndex(client *platformelasticsearch.Client, alias string) *ElasticsearchIndex {
	// 1. 保存平台客户端和稳定别名
	return &ElasticsearchIndex{client: client, alias: strings.TrimSpace(alias)}
}

// CreateIndex 创建带中文、完整拼音和标签分析器的物理索引。
func (e *ElasticsearchIndex) CreateIndex(ctx context.Context, indexName string) error {
	// 1. 构建显式 analysis 和 mapping，禁止未知字段动态扩张
	body, err := json.Marshal(articleIndexDefinition())
	if err != nil {
		return fmt.Errorf("序列化文章搜索索引定义失败: %w", err)
	}

	// 2. 创建新的版本化物理索引
	response, err := (esapi.IndicesCreateRequest{Index: indexName, Body: bytes.NewReader(body)}).Do(ctx, e.transport())
	if err != nil {
		return fmt.Errorf("创建文章搜索索引 %s 失败: %w", indexName, err)
	}
	return checkResponse(response, http.StatusOK)
}

// BulkUpsert 批量新增或覆盖文章搜索文档。
func (e *ElasticsearchIndex) BulkUpsert(ctx context.Context, indexName string, documents []searchdomain.ArticleDocument) error {
	// 1. 空文档集合无需访问 Elasticsearch
	if len(documents) == 0 {
		return nil
	}

	// 2. 组装 Bulk index 元数据和文档 NDJSON
	var body bytes.Buffer
	encoder := json.NewEncoder(&body)
	for _, document := range documents {
		metadata := map[string]any{"index": map[string]any{"_id": strconv.FormatUint(document.ArticleID, 10)}}
		if err := encoder.Encode(metadata); err != nil {
			return fmt.Errorf("序列化文章 %d Bulk 元数据失败: %w", document.ArticleID, err)
		}
		if err := encoder.Encode(document); err != nil {
			return fmt.Errorf("序列化文章 %d 搜索文档失败: %w", document.ArticleID, err)
		}
	}
	return e.executeBulk(ctx, e.resolveIndex(indexName), &body)
}

// DeleteDocuments 按文章 ID 幂等删除搜索文档。
func (e *ElasticsearchIndex) DeleteDocuments(ctx context.Context, indexName string, articleIDs []uint64) error {
	// 1. 空 ID 集合无需访问 Elasticsearch
	if len(articleIDs) == 0 {
		return nil
	}

	// 2. 组装 Bulk delete NDJSON，不存在文档由响应解析视为幂等成功
	var body bytes.Buffer
	encoder := json.NewEncoder(&body)
	for _, articleID := range articleIDs {
		metadata := map[string]any{"delete": map[string]any{"_id": strconv.FormatUint(articleID, 10)}}
		if err := encoder.Encode(metadata); err != nil {
			return fmt.Errorf("序列化文章 %d 删除操作失败: %w", articleID, err)
		}
	}
	return e.executeBulk(ctx, e.resolveIndex(indexName), &body)
}

// Refresh 刷新物理索引使已写文档可查询。
func (e *ElasticsearchIndex) Refresh(ctx context.Context, indexName string) error {
	// 1. 请求 Elasticsearch 刷新指定物理索引
	response, err := (esapi.IndicesRefreshRequest{Index: []string{indexName}}).Do(ctx, e.transport())
	if err != nil {
		return fmt.Errorf("刷新文章搜索索引 %s 失败: %w", indexName, err)
	}
	return checkResponse(response, http.StatusOK)
}

// Count 返回物理索引中的文章文档数量。
func (e *ElasticsearchIndex) Count(ctx context.Context, indexName string) (uint64, error) {
	// 1. 查询指定索引文档总数
	response, err := (esapi.CountRequest{Index: []string{indexName}}).Do(ctx, e.transport())
	if err != nil {
		return 0, fmt.Errorf("统计文章搜索索引 %s 失败: %w", indexName, err)
	}
	body, err := responseBody(response, http.StatusOK)
	if err != nil {
		return 0, err
	}
	var result struct {
		Count uint64 `json:"count"` // Elasticsearch 返回的文档数量
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("解析文章搜索索引数量失败: %w", err)
	}
	return result.Count, nil
}

// SwitchAlias 原子地把稳定别名切换到新物理索引。
func (e *ElasticsearchIndex) SwitchAlias(ctx context.Context, alias, newIndex string) error {
	// 1. 使用调用方别名或 Adapter 默认别名
	alias = strings.TrimSpace(alias)
	if alias == "" {
		alias = e.alias
	}
	body, err := json.Marshal(map[string]any{
		"actions": []any{
			map[string]any{"remove": map[string]any{"index": "*", "alias": alias, "must_exist": false}},
			map[string]any{"add": map[string]any{"index": newIndex, "alias": alias, "is_write_index": true}},
		},
	})
	if err != nil {
		return fmt.Errorf("序列化文章搜索别名切换请求失败: %w", err)
	}

	// 2. 原子提交别名移除和新增动作
	response, err := (esapi.IndicesUpdateAliasesRequest{Body: bytes.NewReader(body)}).Do(ctx, e.transport())
	if err != nil {
		return fmt.Errorf("切换文章搜索别名 %s 失败: %w", alias, err)
	}
	return checkResponse(response, http.StatusOK)
}

// DeleteIndex 删除指定物理索引，索引不存在时保持幂等成功。
func (e *ElasticsearchIndex) DeleteIndex(ctx context.Context, indexName string) error {
	// 1. 删除指定索引并允许索引不存在
	ignoreUnavailable := true
	response, err := (esapi.IndicesDeleteRequest{
		Index:             []string{indexName},
		IgnoreUnavailable: &ignoreUnavailable,
	}).Do(ctx, e.transport())
	if err != nil {
		return fmt.Errorf("删除文章搜索索引 %s 失败: %w", indexName, err)
	}
	return checkResponse(response, http.StatusOK, http.StatusNotFound)
}

// Search 按固定字段权重查询文章并解析高亮结果。
func (e *ElasticsearchIndex) Search(ctx context.Context, query searchdomain.SearchQuery) (*searchdomain.SearchResult, error) {
	// 1. 构建固定相关性查询和原始字段高亮
	from := int((query.Page - 1) * query.PageSize)
	size := int(query.PageSize)
	requestBody := map[string]any{
		"from":             from,
		"size":             size,
		"track_total_hits": true,
		"query": map[string]any{
			"bool": map[string]any{
				"minimum_should_match": 1,
				"should": []any{
					map[string]any{"match": map[string]any{"title": map[string]any{"query": query.Keyword, "boost": 5}}},
					map[string]any{"match": map[string]any{"title.pinyin": map[string]any{"query": query.Keyword, "operator": "and", "boost": 3}}},
					map[string]any{"match": map[string]any{"tags": map[string]any{"query": query.Keyword, "boost": 3}}},
					map[string]any{"match": map[string]any{"title.pinyin_initial": map[string]any{"query": query.Keyword, "boost": 2}}},
					map[string]any{"match": map[string]any{"content": map[string]any{"query": query.Keyword, "boost": 1}}},
				},
			},
		},
		"highlight": map[string]any{
			"pre_tags":  []string{"<em>"},
			"post_tags": []string{"</em>"},
			"fields": map[string]any{
				"title":   map[string]any{"number_of_fragments": 0},
				"content": map[string]any{"fragment_size": 160, "number_of_fragments": 1},
			},
		},
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("序列化文章搜索查询失败: %w", err)
	}

	// 2. 执行分页查询并读取响应
	response, err := (esapi.SearchRequest{
		Index: []string{e.alias}, Body: bytes.NewReader(body),
	}).Do(ctx, e.transport())
	if err != nil {
		return nil, fmt.Errorf("查询文章搜索索引失败: %w", err)
	}
	responseBytes, err := responseBody(response, http.StatusOK)
	if err != nil {
		return nil, err
	}

	// 3. 解析命中来源字段和可选高亮片段
	var searchResponse elasticsearchSearchResponse
	if err := json.Unmarshal(responseBytes, &searchResponse); err != nil {
		return nil, fmt.Errorf("解析文章搜索响应失败: %w", err)
	}
	hits := make([]searchdomain.SearchHit, 0, len(searchResponse.Hits.Hits))
	for _, hit := range searchResponse.Hits.Hits {
		articleID := hit.Source.ArticleID
		if articleID == 0 {
			articleID, _ = strconv.ParseUint(hit.ID, 10, 64)
		}
		hits = append(hits, searchdomain.SearchHit{
			ArticleID:      articleID,
			Title:          hit.Source.Title,
			TitleHighlight: firstString(hit.Highlight.Title),
			Summary:        firstString(hit.Highlight.Content),
			Tags:           hit.Source.Tags,
		})
	}
	return &searchdomain.SearchResult{Hits: hits, Total: searchResponse.Hits.Total.Value}, nil
}

// executeBulk 执行 Bulk 请求并识别部分失败。
func (e *ElasticsearchIndex) executeBulk(ctx context.Context, indexName string, body io.Reader) error {
	// 1. 执行 Bulk 请求并读取整体响应
	response, err := (esapi.BulkRequest{Index: indexName, Body: body}).Do(ctx, e.transport())
	if err != nil {
		return fmt.Errorf("执行文章搜索 Bulk 请求失败: %w", err)
	}
	responseBytes, err := responseBody(response, http.StatusOK)
	if err != nil {
		return err
	}

	// 2. 解析条目级失败，delete 404 视为幂等成功
	var bulkResponse struct {
		Errors bool                               `json:"errors"` // 是否包含失败条目
		Items  []map[string]bulkOperationResponse `json:"items"`  // 每个 Bulk 操作的响应
	}
	if err := json.Unmarshal(responseBytes, &bulkResponse); err != nil {
		return fmt.Errorf("解析文章搜索 Bulk 响应失败: %w", err)
	}
	if !bulkResponse.Errors {
		return nil
	}
	for _, item := range bulkResponse.Items {
		for action, operation := range item {
			if operation.Status >= 200 && operation.Status < 300 {
				continue
			}
			if action == "delete" && operation.Status == http.StatusNotFound {
				continue
			}
			return fmt.Errorf("文章搜索 Bulk %s 文档 %s 失败，状态码 %d: %v", action, operation.ID, operation.Status, operation.Error)
		}
	}
	return nil
}

// resolveIndex 解析写入目标，空值表示使用稳定别名。
func (e *ElasticsearchIndex) resolveIndex(indexName string) string {
	// 1. 增量同步未指定物理索引时写入稳定别名
	if strings.TrimSpace(indexName) == "" {
		return e.alias
	}
	return strings.TrimSpace(indexName)
}

// transport 返回官方客户端的请求传输层。
func (e *ElasticsearchIndex) transport() esapi.Transport {
	// 1. 返回低层 API 共用的传输实现
	return e.client.API.Transport
}

// articleIndexDefinition 返回文章搜索索引显式配置。
func articleIndexDefinition() map[string]any {
	// 1. 定义完整拼音 tokenizer、中文字段和严格 mapping
	return map[string]any{
		"settings": map[string]any{
			"analysis": map[string]any{
				"tokenizer": map[string]any{
					"article_pinyin_tokenizer": map[string]any{
						"type": "pinyin", "keep_first_letter": false, "keep_full_pinyin": true,
						"keep_joined_full_pinyin": true, "keep_original": false,
						"lowercase": true, "remove_duplicated_term": true,
					},
					"article_pinyin_initial_tokenizer": map[string]any{
						"type": "pinyin", "keep_first_letter": true, "keep_separate_first_letter": false,
						"limit_first_letter_length": 16, "keep_full_pinyin": false,
						"keep_joined_full_pinyin": false, "keep_original": false,
						"lowercase": true, "remove_duplicated_term": true,
					},
				},
				"analyzer": map[string]any{
					"article_pinyin":         map[string]any{"tokenizer": "article_pinyin_tokenizer"},
					"article_pinyin_initial": map[string]any{"tokenizer": "article_pinyin_initial_tokenizer"},
					"article_tag": map[string]any{
						"type": "custom", "tokenizer": "ik_max_word", "filter": []string{"lowercase"},
					},
				},
			},
		},
		"mappings": map[string]any{
			"dynamic": "strict",
			"properties": map[string]any{
				"article_id": map[string]any{"type": "unsigned_long"},
				"title": map[string]any{
					"type": "text", "analyzer": "ik_max_word", "search_analyzer": "ik_smart",
					"fields": map[string]any{
						"pinyin":         map[string]any{"type": "text", "analyzer": "article_pinyin", "search_analyzer": "article_pinyin"},
						"pinyin_initial": map[string]any{"type": "text", "analyzer": "article_pinyin_initial", "search_analyzer": "article_pinyin_initial"},
					},
				},
				"content":      map[string]any{"type": "text", "analyzer": "ik_max_word", "search_analyzer": "ik_smart"},
				"tags":         map[string]any{"type": "text", "analyzer": "article_tag", "search_analyzer": "article_tag"},
				"updated_time": map[string]any{"type": "date"},
			},
		},
	}
}

// checkResponse 校验 Elasticsearch HTTP 状态并关闭响应体。
func checkResponse(response *esapi.Response, allowedStatuses ...int) error {
	// 1. 读取响应体以便复用 HTTP 连接
	_, err := responseBody(response, allowedStatuses...)
	return err
}

// responseBody 读取 Elasticsearch 响应并校验允许状态码。
func responseBody(response *esapi.Response, allowedStatuses ...int) ([]byte, error) {
	// 1. 校验响应对象并确保资源释放
	if response == nil {
		return nil, errors.New("Elasticsearch 返回空响应")
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("读取 Elasticsearch 响应失败: %w", err)
	}

	// 2. 仅接受调用方声明的 HTTP 状态码
	for _, status := range allowedStatuses {
		if response.StatusCode == status {
			return body, nil
		}
	}
	return nil, fmt.Errorf("Elasticsearch 请求失败，状态码 %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
}

// firstString 返回字符串切片的首个值，不存在时返回空字符串。
func firstString(values []string) string {
	// 1. 高亮不存在时返回稳定空值
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// bulkOperationResponse 表示单个 Bulk 操作响应。
type bulkOperationResponse struct {
	ID     string         `json:"_id"`    // Elasticsearch 文档 ID
	Status int            `json:"status"` // 操作 HTTP 状态码
	Error  map[string]any `json:"error"`  // 条目失败详情，成功时为空
}

// elasticsearchSearchResponse 表示文章搜索所需的最小响应结构。
type elasticsearchSearchResponse struct {
	Hits struct {
		Total struct {
			Value uint64 `json:"value"` // 精确命中文档总数
		} `json:"total"` // 命中总数对象
		Hits []struct {
			ID        string                       `json:"_id"`     // Elasticsearch 文档 ID
			Source    searchdomain.ArticleDocument `json:"_source"` // 搜索文档来源字段
			Highlight struct {
				Title   []string `json:"title"`   // 原始标题高亮片段
				Content []string `json:"content"` // 正文纯文本高亮摘要
			} `json:"highlight"` // 可选高亮结果
		} `json:"hits"` // 当前页命中文档
	} `json:"hits"` // Elasticsearch hits 响应
}

// NewVersionedIndexName 根据当前时间生成版本化物理索引名称。
func NewVersionedIndexName(alias string, now time.Time) string {
	// 1. 使用 UTC 时间避免部署节点时区造成命名差异
	return fmt.Sprintf("%s_%s", strings.TrimSpace(alias), now.UTC().Format("20060102150405"))
}
