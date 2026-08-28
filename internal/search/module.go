package search

import (
	platformconfig "blog/internal/platform/config"
	platformelasticsearch "blog/internal/platform/elasticsearch"
	searchapp "blog/internal/search/app"
	searchinfra "blog/internal/search/infra"
	searchhttp "blog/internal/search/interfaces/http"
)

// Module 表示 Search 上下文对组合根公开的在线查询能力。
type Module struct {
	Application *searchapp.Service  // 文章搜索 Application Facade
	HTTP        *searchhttp.Handler // 前台文章搜索 HTTP Adapter
}

// NewModule 创建 Search 在线查询模块。
func NewModule(client *platformelasticsearch.Client, cfg platformconfig.Elasticsearch) *Module {
	// 1. 创建 Elasticsearch 查询 Adapter 和应用服务
	index := searchinfra.NewElasticsearchIndex(client, cfg.GetIndexAlias())
	application := searchapp.NewService(index)

	// 2. 暴露前台 HTTP Adapter
	return &Module{Application: application, HTTP: searchhttp.NewHandler(application)}
}
