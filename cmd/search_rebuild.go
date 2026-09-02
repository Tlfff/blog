package cmd

import (
	platformbootstrap "blog/internal/platform/bootstrap"
	platformconfig "blog/internal/platform/config"
	searchapp "blog/internal/search/app"
	searchinfra "blog/internal/search/infra"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

const searchRebuildBatchSize = 200 // 全量重建单批文章数量

var searchRebuildCmd = &cobra.Command{
	Use:   "search-rebuild",
	Short: "从 MySQL 全量重建文章搜索索引",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. 加载搜索和数据库配置
		cfg, err := platformconfig.LoadConfig("config/config.yaml")
		if err != nil {
			return fmt.Errorf("加载搜索重建配置失败: %w", err)
		}

		// 2. 初始化全量重建所需的 MySQL 和 Elasticsearch 资源
		resources, err := platformbootstrap.NewResources(cfg, platformbootstrap.ResourceOptions{
			MySQL: true, Elasticsearch: true,
		})
		if err != nil {
			return fmt.Errorf("初始化搜索重建资源失败: %w", err)
		}
		defer closeResources(resources)

		// 3. 组装 Search 自有文章数据源和统一文档转换能力
		index := searchinfra.NewElasticsearchIndex(resources.Elasticsearch, cfg.Elasticsearch.GetIndexAlias())
		factory := searchapp.NewDocumentFactory(searchinfra.NewMarkdownExtractor())
		rebuild := searchapp.NewRebuildService(
			searchinfra.NewArticleSource(resources.MySQL),
			index,
			factory,
			cfg.Elasticsearch.GetIndexAlias(),
			searchRebuildBatchSize,
		)

		// 4. 构建新索引并在校验通过后切换别名
		indexName, err := rebuild.Rebuild(cmd.Context())
		if err != nil {
			return fmt.Errorf("全量重建文章搜索索引失败: %w", err)
		}
		log.Printf("文章搜索索引重建完成，当前物理索引=%s", indexName)
		return nil
	},
}

// init 注册文章搜索全量重建子命令。
func init() {
	// 1. 把一次性重建入口注册到根命令
	rootCmd.AddCommand(searchRebuildCmd)
}
