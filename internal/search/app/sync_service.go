package app

import (
	searchdomain "blog/internal/search/domain"
	"context"
	"fmt"
)

const activeIndexAlias = "" // 空索引名表示增量同步写入稳定别名

// SyncService 根据文章行变更维护 Elasticsearch 投影。
type SyncService struct {
	writer  searchdomain.IndexWriter // Elasticsearch 索引写入 Port
	factory *DocumentFactory         // 搜索文档工厂
}

// NewSyncService 创建文章搜索增量同步服务。
func NewSyncService(writer searchdomain.IndexWriter, factory *DocumentFactory) *SyncService {
	// 1. 保存索引写入和文档转换依赖
	return &SyncService{writer: writer, factory: factory}
}

// HandleChanges 按原始顺序处理一个 Canal 批次中的文章变更。
func (s *SyncService) HandleChanges(ctx context.Context, changes []searchdomain.ArticleChange) error {
	// 1. 按原始顺序逐条执行幂等索引操作
	for _, change := range changes {
		if err := s.HandleChange(ctx, change); err != nil {
			return err
		}
	}
	return nil
}

// HandleChange 根据文章状态和搜索字段变化决定 upsert、delete 或忽略。
func (s *SyncService) HandleChange(ctx context.Context, change searchdomain.ArticleChange) error {
	// 1. 校验同步依赖
	if s == nil || s.writer == nil || s.factory == nil {
		return searchdomain.ErrSearchUnavailable
	}

	// 2. 根据事件类型和公开状态决定索引操作
	switch change.Type {
	case searchdomain.ChangeTypeInsert:
		// 新增的草稿或已删除文章不进入搜索索引
		if !change.After.IsPublished() {
			return nil
		}
		// 新增已发表文章，写入 ES
		return s.upsert(ctx, change.After)
	case searchdomain.ChangeTypeUpdate:
		// 更新后的文章不再公开
		if !change.After.IsPublished() {
			// 更新前后都未发表，且状态未变化，不需要操作 ES
			if !change.Before.IsPublished() && !change.ChangedFields["status"] {
				return nil
			}
			// 无法获取文章 ID，无法执行删除
			if change.Before.ID == 0 && change.After.ID == 0 {
				return nil
			}
			articleID := change.After.ID
			if articleID == 0 {
				articleID = change.Before.ID
			}
			// 文章转为草稿或已删除状态，从 ES 删除
			return s.writer.DeleteDocuments(ctx, activeIndexAlias, []uint64{articleID})
		}
		// 只有浏览量、点赞数等非搜索字段变化时忽略
		if !hasSearchFieldChanged(change.ChangedFields) {
			return nil
		}
		// 已发表文章的搜索字段变化，覆盖 ES 文档
		return s.upsert(ctx, change.After)
	case searchdomain.ChangeTypeDelete:
		// 物理删除事件缺少文章 ID，无法操作 ES
		if change.Before.ID == 0 {
			return nil
		}
		// MySQL 文章被物理删除，同步删除 ES 文档
		return s.writer.DeleteDocuments(ctx, activeIndexAlias, []uint64{change.Before.ID})
	default:
		return searchdomain.ErrChangeTypeInvalid
	}
}

// upsert 构建并写入单篇文章搜索文档。
func (s *SyncService) upsert(ctx context.Context, article searchdomain.SourceArticle) error {
	// 1. 使用统一工厂构建搜索文档
	document, err := s.factory.Build(ctx, article)
	if err != nil {
		return err
	}

	// 2. 通过稳定别名幂等覆盖文章文档
	if err := s.writer.BulkUpsert(ctx, activeIndexAlias, []searchdomain.ArticleDocument{document}); err != nil {
		return fmt.Errorf("同步文章 %d 搜索文档失败: %w", article.ID, err)
	}
	return nil
}

// hasSearchFieldChanged 判断 UPDATE 是否影响公开搜索投影。
func hasSearchFieldChanged(changedFields map[string]bool) bool {
	// 1. 标题、正文、标签和状态任一变化都需要重新评估索引
	for _, field := range []string{"title", "content", "tags", "status"} {
		if changedFields[field] {
			return true
		}
	}
	return false
}
