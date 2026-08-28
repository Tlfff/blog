package bootstrap

import (
	article "blog/internal/article"
	articleapp "blog/internal/article/app"
	articleinfra "blog/internal/article/infra"
	comment "blog/internal/comment"
	like "blog/internal/like"
	notification "blog/internal/notification"
	platformconfig "blog/internal/platform/config"
	platformtransaction "blog/internal/platform/transaction"
	search "blog/internal/search"
	user "blog/internal/user"
)

// Application 汇总模块化单体的六个业务上下文。
type Application struct {
	User         *user.Module         // User 上下文模块
	Article      *article.Module      // Article 上下文模块
	Comment      *comment.Module      // Comment 上下文模块
	Like         *like.Module         // Like 上下文模块
	Notification *notification.Module // Notification 上下文模块
	Search       *search.Module       // Search 上下文模块，当前入口未初始化 ES 时为空
}

// NewApplication 按上下文依赖顺序组装模块化单体。
func NewApplication(resources *Resources, cfg *platformconfig.Config) (*Application, error) {
	// 1. 创建共享本地事务协调器
	tx, err := platformtransaction.NewManager(resources.MySQL)
	if err != nil {
		return nil, err
	}

	// 2. 创建 User 上下文
	userModule := user.NewModule(resources.MySQL, resources.Redis, resources.OSS, cfg)

	// 3. 预先创建 Article 统计 Facade，供 Comment 和 Like 参加同一事务
	articleStatistics := articleapp.NewStatisticsService(articleinfra.NewArticleStatistics(resources.MySQL))

	// 4. 创建 Comment 和 Like 上下文
	commentModule := comment.NewModule(resources.MySQL, resources.Redis, articleStatistics, tx)
	likeModule := like.NewModule(
		resources.MySQL,
		resources.Redis,
		resources.Kafka,
		articleStatistics,
		commentModule.LikeProjection,
		tx,
	)

	// 5. 创建 Article 上下文，并注入 User/Like 本地 Facade
	articleModule := article.NewModule(
		resources.MySQL,
		resources.Redis,
		resources.Kafka,
		resources.OSS,
		cfg,
		userModule.Application,
		likeModule.Application,
		articleStatistics,
	)

	// 6. 创建 Notification 上下文，并注入 Article/User 本地 Facade
	notificationModule := notification.NewModule(
		resources.MongoDB,
		articleModule.Application,
		userModule.Application,
	)

	// 7. 当前入口需要 Elasticsearch 时创建 Search 在线查询上下文
	var searchModule *search.Module
	if resources.Elasticsearch != nil {
		searchModule = search.NewModule(resources.Elasticsearch, cfg.Elasticsearch)
	}

	return &Application{
		User: userModule, Article: articleModule, Comment: commentModule,
		Like: likeModule, Notification: notificationModule, Search: searchModule,
	}, nil
}
