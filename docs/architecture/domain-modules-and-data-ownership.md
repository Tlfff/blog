# 领域模块与数据所有权矩阵

本文档建立阶段一重构的三个领域模块边界，并固化当前数据归属与目标归属。阶段一允许共享 MySQL/Redis/MongoDB/MinIO/Kafka，但代码层不得继续跨域直接读写数据表或调用对方 Repository。

## 1. 领域模块边界

| 模块 | 职责 | 主要聚合/实体 |
| --- | --- | --- |
| Identity | 用户、认证、会话、角色、个人资料、头像 | `User`、登录会话、改密凭证 |
| Content | 文章生命周期、文章查询、文章图片 | `Article`、文章状态、图片对象 |
| Community | 评论、回复、文章点赞、评论点赞、浏览、统计、热榜 | `Comment`、`ArticleLike`、`CommentLike`、`ArticleViewHistory` |
| Community.Notification | 通知生成、查询、未读状态、异步消费 | `Notification` |

通知在阶段一作为 Community 模块内部子模块实现，阶段二随 Community Service 一起部署，不单独拆服务。

## 2. 数据所有权矩阵

### 2.1 MySQL

| 表 | 当前写入方 | 目标所有权 | 说明 |
| --- | --- | --- | --- |
| `users` | UserRepository | Identity | 用户账号、角色、资料、登录信息 |
| `articles` 基础字段 | ArticleRepository | Content | `author_id/title/content/tags/status` |
| `articles` 统计字段 | ArticleLike/ViewHistory/Comment 相关 Repository | Community（统计读模型） | `view_count/like_count/comment_count` 当前由 Community 侧逻辑直接写，阶段一需收敛为 Community 拥有的统计能力 |
| `comments` | CommentRepository | Community | 评论与回复 |
| `article_likes` | ArticleLikeRepository | Community | 文章点赞 |
| `comment_likes` | CommentLikeRepository | Community | 评论点赞 |
| `article_view_histories` | ArticleViewHistoryRepository | Community | 浏览历史 |

### 2.2 MongoDB

| 集合 | 当前写入方 | 目标所有权 |
| --- | --- | --- |
| `notifications` | NotificationRepository | Community.Notification |

### 2.3 Redis

| Key 范围 | 当前写入方 | 目标所有权 |
| --- | --- | --- |
| `auth:token:*`、`auth:user-tokens:*` | TokenAuth | Identity |
| `user:password-change:*` | UserService | Identity |
| `like:article:*`、`like:comment:*` | ArticleLike/CommentLike Service | Community |
| `lock:like:article:*`、`lock:like:comment:*`、`lock:init:like:*` | 点赞 Service | Community |
| `rank:article:hot`、`lock:init:rank:article:hot` | ArticleRankService | Community |
| `article:info:*`、`lock:init:article:info:*` | Article/点赞 Service | Community（统计读模型） |
| `hmac:nonce:*` | gRPC 拦截器 | 共享平台/Infrastructure |

### 2.4 MinIO

| Object 前缀 | 当前写入方 | 目标所有权 |
| --- | --- | --- |
| `avatar/{user_id}/*` | UserService | Identity |
| `article/temp/*` | ArticleImageService | Content |
| `article/{article_id}/*` | ArticleService | Content |

### 2.5 Kafka

| Topic | 当前生产者/消费者 | 目标所有权 |
| --- | --- | --- |
| `notification` | ArticleLikeService 生产、MQ 消费 | Community.Notification |
| `view_history` | ArticleViewHistoryService 生产/消费 | Community |
| `dead_letter` | Kafka 基础设施 | 共享平台 |

## 3. 当前跨域数据访问清单

| 编号 | 来源 | 目标 | 具体位置 | 处置方向 |
| --- | --- | --- | --- | --- |
| D1 | Content Repository | Identity 数据表 | `ArticleRepository` JOIN `users` 返回昵称/头像/IP | 通过 Identity 查询 Port 或 gRPC 获取用户公开信息 |
| D2 | Community Repository | Identity 数据表 | `CommentRepository` JOIN `users` 返回发评人/被回复人资料 | 通过 Identity 查询 Port 获取 |
| D3 | Community Service | Identity Repository | `ArticleLikeService` 持有 `UserRepository` 组装通知发送方 | 通过 Identity 查询 Port 获取 |
| D4 | Community Service | Content Repository/表 | `CommentService` 持有 `ArticleRepository` 并更新 `articles.comment_count` | 评论统计与文章统计通过 Community 统计能力/读模型提供 |
| D5 | Community Repository | Content 统计列 | `ArticleLikeRepository.UpdateArticleLikeCountDelta` 写 `articles.like_count` | 点赞数由 Community 统计读模型维护 |
| D6 | Community Repository | Content 统计列 | `ArticleViewHistoryRepository.IncrementViewCount` 写 `articles.view_count` | 浏览量由 Community 统计读模型维护 |
| D7 | Content Service | Community Service | `ArticleService` 依赖 `ArticleLikeService` 查询点赞状态 | 通过 Community 统计查询 Port 获取 |

阶段一后续任务（3.x-5.x）必须以本矩阵为目标收敛上述跨域访问。
