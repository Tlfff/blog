# 阶段二数据所有权执行

共享 MySQL 阶段，三个服务通过“代码只初始化自己的 Repository + 只读查询 Port”来执行逻辑数据所有权。

## 1. 服务访问范围

| 服务 | 写/全部访问 | 只读访问 |
| --- | --- | --- |
| Identity | `users`、Redis `auth:*`/`user:*`、MinIO `avatar/*` | 无 |
| Content | `articles` 基础字段、MinIO `article/*` | Identity 作者信息（gRPC）、Community 点赞状态（gRPC） |
| Community | `comments`、`article_likes`、`comment_likes`、`article_view_histories`、MongoDB `notifications`、Redis `like:*`/`rank:*` | Content 文章基本信息（gRPC）、Identity 用户公开信息（gRPC） |
| 统一入口 | 无数据库/中间件写权限 | Redis 仅用于会话鉴权与 nonce |

## 2. 代码约束

- `services/identity` 只初始化 `UserRepository`。
- `services/content` 只初始化 `ArticleRepository` 与文章图片存储。
- `services/community` 只初始化评论、点赞、浏览、通知相关 Repository。
- 统一入口不再连接 MySQL、MongoDB、Kafka、MinIO。
- 跨域数据通过内部 gRPC 只读接口获取：`IdentityService.GetUserBasicInfo`、`ContentService.GetArticleInfo`、`CommunityService.IsUserLikedArticle`。

## 3. 共享表统计列

物理拆库前，Community 仍通过共享 MySQL 的 `articles` 统计列维护点赞/评论/浏览计数，但只能通过 Community 自己的统计适配器更新，不允许直接调用 Content Repository。逻辑所有权稳定并通过校验后，再执行物理迁移。
