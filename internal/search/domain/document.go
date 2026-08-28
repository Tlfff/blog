package domain

import "time"

// ArticleSourceStatus 表示搜索来源文章的生命周期状态。
type ArticleSourceStatus int8

const (
	ArticleStatusDeleted   ArticleSourceStatus = 1 // 已删除，不应存在于公开搜索索引
	ArticleStatusDraft     ArticleSourceStatus = 2 // 草稿，不应存在于公开搜索索引
	ArticleStatusPublished ArticleSourceStatus = 3 // 已发表，应存在于公开搜索索引
)

// ArticleDocument 表示 Elasticsearch 中可重建的文章搜索文档。
type ArticleDocument struct {
	ArticleID   uint64    `json:"article_id"`   // 文章唯一标识
	Title       string    `json:"title"`        // 文章原始标题
	Content     string    `json:"content"`      // 已移除 Markdown、图片和代码的正文纯文本
	Tags        []string  `json:"tags"`         // 去空格、排除空值并按英文大小写去重后的标签
	UpdatedTime time.Time `json:"updated_time"` // 文章最后更新时间
}

// SearchQuery 表示文章搜索查询条件。
type SearchQuery struct {
	Keyword  string // 去除首尾空格后的搜索关键词
	Page     uint64 // 页码，从 1 开始
	PageSize uint64 // 每页数量，范围 10 至 20
}

// SearchHit 表示搜索引擎返回的单篇文章命中。
type SearchHit struct {
	ArticleID      uint64   // 文章唯一标识
	Title          string   // 文章原始标题
	TitleHighlight string   // 原始标题高亮，仅拼音命中时可以为空
	Summary        string   // 正文纯文本高亮摘要，未命中正文时可以为空
	Tags           []string // 规范化标签列表
}

// SearchResult 表示文章搜索分页结果。
type SearchResult struct {
	Hits  []SearchHit // 当前页搜索命中
	Total uint64      // 符合条件的文章总数
}

// SourceArticle 表示构建搜索文档所需的最小文章数据。
type SourceArticle struct {
	ID          uint64              // 文章唯一标识
	Title       string              // 文章原始标题
	Content     string              // Markdown 正文
	Tags        string              // 英文逗号分隔的标签字符串
	Status      ArticleSourceStatus // 来源文章状态：1-已删除；2-草稿；3-已发表
	UpdatedTime time.Time           // 文章最后更新时间
}

// ChangeType 表示文章行级变更类型。
type ChangeType int8

const (
	ChangeTypeInsert ChangeType = 1 // 新增文章行
	ChangeTypeUpdate ChangeType = 2 // 更新文章行
	ChangeTypeDelete ChangeType = 3 // 删除文章行
)

// ArticleChange 表示从 Canal 行变更转换出的搜索业务事件。
type ArticleChange struct {
	Type          ChangeType      // 行变更类型：1-新增；2-更新；3-删除
	Before        SourceArticle   // 变更前文章数据，新增事件为空
	After         SourceArticle   // 变更后文章数据，删除事件为空
	ChangedFields map[string]bool // UPDATE 事件实际发生变化的字段集合
}

// IsPublished 判断来源文章是否已发表。
func (a SourceArticle) IsPublished() bool {
	// 1. 只有已发表状态允许进入公开搜索索引
	return a.Status == ArticleStatusPublished
}
