package domain

import "context"

// IndexSearcher 定义文章搜索查询能力。
type IndexSearcher interface {
	// Search 按相关性查询文章并返回高亮结果。
	Search(ctx context.Context, query SearchQuery) (*SearchResult, error)
}

// IndexWriter 定义文章搜索索引写入和管理能力。
type IndexWriter interface {
	// CreateIndex 创建带有显式 mapping 的物理索引。
	CreateIndex(ctx context.Context, indexName string) error
	// BulkUpsert 批量新增或覆盖文章文档。
	BulkUpsert(ctx context.Context, indexName string, documents []ArticleDocument) error
	// DeleteDocuments 按文章 ID 幂等删除文档。
	DeleteDocuments(ctx context.Context, indexName string, articleIDs []uint64) error
	// Refresh 刷新物理索引使已写文档可查询。
	Refresh(ctx context.Context, indexName string) error
	// Count 返回物理索引中的文章文档数量。
	Count(ctx context.Context, indexName string) (uint64, error)
	// SwitchAlias 原子地把稳定别名切换到新物理索引。
	SwitchAlias(ctx context.Context, alias, newIndex string) error
	// DeleteIndex 删除指定物理索引，索引不存在时保持幂等成功。
	DeleteIndex(ctx context.Context, indexName string) error
}

// ArticleSource 定义全量重建所需的只读文章数据源。
type ArticleSource interface {
	// ListPublishedAfter 按文章 ID 游标分批读取已发表文章。
	ListPublishedAfter(ctx context.Context, lastID uint64, limit int) ([]SourceArticle, error)
	// CountPublished 统计当前已发表文章数量。
	CountPublished(ctx context.Context) (uint64, error)
}

// TextExtractor 定义 Markdown 正文到可搜索纯文本的转换能力。
type TextExtractor interface {
	// Extract 提取正文中的普通可读文本。
	Extract(ctx context.Context, markdown string) (string, error)
}
