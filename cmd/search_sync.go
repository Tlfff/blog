package cmd

import (
	platformbootstrap "blog/internal/platform/bootstrap"
	platformcanal "blog/internal/platform/canal"
	platformconfig "blog/internal/platform/config"
	searchapp "blog/internal/search/app"
	searchinfra "blog/internal/search/infra"
	searchcanal "blog/internal/search/interfaces/canal"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var searchSyncCmd = &cobra.Command{
	Use:   "search-sync",
	Short: "启动 Canal 到 Elasticsearch 的文章搜索增量同步",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. 加载并校验搜索同步配置
		cfg, err := platformconfig.LoadConfig("config/config.yaml")
		if err != nil {
			return fmt.Errorf("加载搜索同步配置失败: %w", err)
		}
		if err := cfg.Canal.Validate(); err != nil {
			return err
		}

		// 2. 初始化同步进程仅需要的 Elasticsearch 资源
		resources, err := platformbootstrap.NewResources(cfg, platformbootstrap.ResourceOptions{Elasticsearch: true})
		if err != nil {
			return fmt.Errorf("初始化搜索同步资源失败: %w", err)
		}
		defer closeResources(resources)

		// 3. 组装文档转换、索引写入和 Canal 协议 Adapter
		index := searchinfra.NewElasticsearchIndex(resources.Elasticsearch, cfg.Elasticsearch.GetIndexAlias())
		factory := searchapp.NewDocumentFactory(searchinfra.NewMarkdownExtractor())
		syncService := searchapp.NewSyncService(index, factory)
		handler := searchcanal.NewHandler(syncService, cfg.Database.DBName, "articles")
		client, err := platformcanal.NewClient(cfg.Canal, handler)
		if err != nil {
			return fmt.Errorf("创建 Canal Client 失败: %w", err)
		}
		defer func() {
			if err := client.Close(); err != nil {
				log.Printf("[WARN] 关闭 Canal Client 失败: %v", err)
			}
		}()

		// 4. 运行单实例同步循环并响应退出信号
		ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()
		log.Printf("文章搜索增量同步开始运行，destination=%s", cfg.Canal.Destination)
		if err := client.Run(ctx); err != nil {
			return fmt.Errorf("文章搜索增量同步退出: %w", err)
		}
		return nil
	},
}

// init 注册文章搜索增量同步子命令。
func init() {
	// 1. 把长期运行的同步入口注册到根命令
	rootCmd.AddCommand(searchSyncCmd)
}
