package app

import (
	searchdomain "blog/internal/search/domain"
	"context"
	"fmt"
	"strings"
	"time"
)

const defaultRebuildBatchSize = 200 // 默认全量重建单批文章数量

// RebuildService 负责从 MySQL 全量重建文章搜索索引。
type RebuildService struct {
	source    searchdomain.ArticleSource // 已发表文章只读数据源
	writer    searchdomain.IndexWriter   // Elasticsearch 索引管理 Port
	factory   *DocumentFactory           // 统一搜索文档工厂
	alias     string                     // 文章搜索稳定索引别名
	batchSize int                        // 单批读取和写入文章数量
	now       func() time.Time           // 生成版本化索引名的时钟
}

// NewRebuildService 创建文章搜索全量重建服务。
//
// 参数说明：
//   - source：已发表文章只读数据源。
//   - writer：Elasticsearch 索引写入和管理能力。
//   - factory：统一搜索文档工厂。
//   - alias：文章搜索稳定索引别名。
//   - batchSize：单批文章数量，小于等于 0 时使用默认值。
func NewRebuildService(
	source searchdomain.ArticleSource,
	writer searchdomain.IndexWriter,
	factory *DocumentFactory,
	alias string,
	batchSize int,
) *RebuildService {
	// 1. 规范化索引别名和批次大小
	alias = strings.TrimSpace(alias)
	if batchSize <= 0 {
		batchSize = defaultRebuildBatchSize
	}

	// 2. 保存重建依赖并使用当前时间生成索引版本
	return &RebuildService{
		source: source, writer: writer, factory: factory, alias: alias,
		batchSize: batchSize, now: time.Now,
	}
}

// Rebuild 创建新物理索引、导入已发表文章并原子切换别名。
func (s *RebuildService) Rebuild(ctx context.Context) (string, error) {
	// 1. 校验重建依赖并创建新版本物理索引
	if s == nil || s.source == nil || s.writer == nil || s.factory == nil || s.alias == "" {
		return "", searchdomain.ErrSearchUnavailable
	}
	indexName := newVersionedIndexName(s.alias, s.now())
	if err := s.writer.CreateIndex(ctx, indexName); err != nil {
		return "", fmt.Errorf("创建文章搜索重建索引失败: %w", err)
	}
	switched := false
	defer func() {
		if !switched {
			// 重建请求取消后仍为失败索引清理保留有限时间，避免遗留不可用索引。
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = s.writer.DeleteIndex(cleanupCtx, indexName)
		}
	}()

	// 2. 按文章 ID 游标分批转换并写入已发表文章
	lastID := uint64(0)
	writtenCount := uint64(0)
	for {
		articles, err := s.source.ListPublishedAfter(ctx, lastID, s.batchSize)
		if err != nil {
			return "", err
		}
		if len(articles) == 0 {
			break
		}
		documents := make([]searchdomain.ArticleDocument, 0, len(articles))
		for _, article := range articles {
			document, err := s.factory.Build(ctx, article)
			if err != nil {
				return "", err
			}
			documents = append(documents, document)
			lastID = article.ID
		}
		if err := s.writer.BulkUpsert(ctx, indexName, documents); err != nil {
			return "", fmt.Errorf("批量写入文章搜索重建索引失败: %w", err)
		}
		writtenCount += uint64(len(documents))
	}

	// 3. 刷新并核对实际文档数，防止部分写入被误判成功
	if err := s.writer.Refresh(ctx, indexName); err != nil {
		return "", err
	}
	indexedCount, err := s.writer.Count(ctx, indexName)
	if err != nil {
		return "", err
	}
	if indexedCount != writtenCount {
		return "", fmt.Errorf("文章搜索重建数量不一致: 写入 %d，索引 %d", writtenCount, indexedCount)
	}
	// 4. 数量校验通过后原子切换稳定别名
	if err := s.writer.SwitchAlias(ctx, s.alias, indexName); err != nil {
		return "", err
	}
	switched = true
	return indexName, nil
}

// newVersionedIndexName 根据 UTC 时间生成版本化物理索引名称。
func newVersionedIndexName(alias string, now time.Time) string {
	// 1. 使用秒级 UTC 时间保证名称稳定且便于运维识别
	return fmt.Sprintf("%s_%s", strings.TrimSpace(alias), now.UTC().Format("20060102150405"))
}
