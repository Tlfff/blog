# 当前模块调用矩阵

本文档记录阶段一改造前的实际调用关系，用于识别 Handler 直连 Repository、Service 跨域依赖和基础设施泄漏点。

## 1. 入口层调用

| 入口 | 调用目标 | 说明 |
| --- | --- | --- |
| HTTP Handler（`internal/handler`） | UserAuthService、UserService、ArticleService、ArticleRankService、ArticleImageService、CommentService、ArticleLikeService、CommentLikeService、NotificationService | 无生产代码直连 Repository |
| gRPC Handler（`internal/grpc/handler`） | UserService、ArticleService、CommentService | 二方开放接口 |
| Kafka Consumer（`internal/mq`） | ArticleLikeService、ArticleViewHistoryService | 通知与浏览历史消费 |
| Cron（`internal/cron`） | ArticleRankService | 热榜重建 |

## 2. Service 依赖矩阵

| Service | Repository/Storage | 基础设施 | 跨域 Service 依赖 |
| --- | --- | --- | --- |
| UserAuthService | UserRepository | TokenAuth（Redis） | 无 |
| UserService | UserRepository | Redis、MinIO | 无 |
| ArticleService | ArticleRepository | Redis、MinIO | ArticleLikeService |
| ArticleImageService | 无 | MinIO | 无 |
| ArticleLikeService | ArticleLikeRepository、ArticleRepository、UserRepository | Redis、Kafka、`database.RunTx` | NotificationService |
| ArticleRankService | ArticleRepository | Redis | 无 |
| ArticleViewHistoryService | ArticleViewHistoryRepository | Kafka | 无 |
| CommentService | CommentRepository、ArticleRepository | Redis、`database.RunTx` | 无 |
| CommentLikeService | CommentLikeRepository、CommentRepository | Redis、`database.RunTx` | 无 |
| NotificationService | NotificationRepository（MongoDB） | 无 | 无 |

## 3. Repository 跨表/跨域访问

| Repository | 访问对象 | 问题 |
| --- | --- | --- |
| ArticleRepository | `articles` + `users` | Content 跨域读 Identity 数据 |
| CommentRepository | `comments` + `users` | Community 跨域读 Identity 数据 |
| ArticleLikeRepository | `article_likes` + `articles.like_count` | Community 跨域写 Content 统计列 |
| ArticleViewHistoryRepository | `article_view_histories` + `articles.view_count` | Community 跨域写 Content 统计列 |
| CommentRepository | `comments` + 主楼 `comments.comment_count` | 同域，正常 |
| ArticleRepository | `articles.comment_count`（被 CommentService 调用） | Community 业务通过 Content Repository 写 Content 表 |

## 4. 基础设施泄漏点

| 位置 | 泄漏内容 | 目标 |
| --- | --- | --- |
| `internal/service/*` | 直接依赖 GORM、Redis Client、Kafka Client、MinIO Client、`pkg/database.RunTx` | 收敛到 Infrastructure Adapter，Service 只依赖 Port |
| `internal/service` | `ArticleViewHistoryService` 直接调用 Kafka Producer | 收敛到事件发布 Port |
| `internal/service` | `ArticleLikeService` 直接调用 `database.RunTx` 和 `GetDB()` | 收敛到 Repository/UnitOfWork Port |
| `internal/handler` | 部分 Handler 导入 `internal/model` 业务实体 | Interfaces 层不应直接依赖 Domain 模型 |
| `internal/middleware` | `AdminCheckMiddleware` 导入 `internal/model` 角色常量 | 通过应用/领域接口表达角色权限 |
| `internal/routes` | `AppHandler` 持有 `*redis.Client` | 鉴权中间件依赖收敛到 Interfaces 层 Port |

## 5. 汇总

当前生产 Handler 没有直连 Repository；主要问题集中在 Service 直接依赖基础设施、Content/Community 跨域访问 Repository 或数据表、以及 Interfaces 层混入领域模型。阶段一重构将按 1.5 的四层规则逐项消除。
