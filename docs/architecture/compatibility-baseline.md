# DDD 重构行为基线与迁移清单

## 1. 基线范围

- **行为基线提交**：`70b6558709e587ab1408375d1598b6b9e192e770`
- **当前分支说明**：当前分支仅在该提交之上增加设计文档、OpenSpec 规划文件和工作规范；未将 `feature/DDD` 或 `82aa39e` 的实现作为运行基线。
- **基线运行形态**：同一个 Go 二进制通过 Cobra 提供 `server`、`grpc`、`kafka-consume` 和 `migrate` 子命令。
- **本次文档用途**：固定重构前可观察契约、数据副作用、基础设施资源和现有测试覆盖，作为逐上下文迁移及回归验收的对照清单。
- **不在基线中补齐的内容**：旧代码中未接线的通知类型、评论事件、目标显式校验、事件 Envelope/幂等和参考分支中的微服务/内部 gRPC 能力均不属于本基线行为。

## 2. 运行入口与装配

| 入口 | 代码位置 | 主要职责 | 关键资源 |
| --- | --- | --- | --- |
| `blog server` | `cmd/server.go` | HTTP 服务、路由、中间件、Cron、Kafka Producer | MySQL、MongoDB、Redis、Kafka、MinIO、IP 查询器 |
| `blog grpc` | `cmd/grpc.go` | 对外 gRPC 服务 | MySQL、Redis、User/Article/Comment Service |
| `blog kafka-consume` | `cmd/kafka_consume.go` | Kafka Consumer，处理通知和浏览历史 | MySQL、MongoDB、Redis、Kafka |
| `blog migrate` | `cmd/migrate.go` | 执行 `scripts/mysql` 建库和建表 SQL | MySQL |

`server` 当前直接创建全部 Repository、Service、Handler、Cron 和路由；`grpc` 只装配 gRPC 所需的 User、Article、Comment Service；`kafka-consume` 装配通知和浏览历史消费所需 Service。重构时需要保持子命令职责和资源按需初始化行为，不能因为建立组合根而让所有入口无条件初始化全部客户端。

## 3. HTTP 契约清单

### 3.1 全局处理顺序

HTTP Engine 的全局中间件顺序必须保持为：

```text
LoggerMiddleware
    ↓
GlobalErrorMiddleware
    ↓
路由组中间件
    ↓
Handler
```

- `LoggerMiddleware`：读取或生成 `X-Trace-ID`，写回响应头，记录请求方法、路径、Query、POST Body（最多 1KB）、Authorization、User-Agent、响应状态、耗时和错误。
- `GlobalErrorMiddleware`：执行 `c.Next()` 后读取最后一个 Gin Error，使用 `common.GetCodeByError` 映射业务码，通过统一响应返回；HTTP 状态通常仍为 200，除非测试或其他入口单独模拟状态。
- `MustAuth`：校验 `Authorization: Bearer <token>`，从 Redis Session 获取用户，注入 `currentUser` 与 `currentToken`；失败时 `c.Error`、`Abort`。
- `OptionalAuth`：尝试同样的 Token 校验，失败时不拦截，成功时注入用户和 Token。
- `AdminCheckMiddleware`：要求 `currentUser.Role == model.RoleAdmin`，否则返回 `ErrForbidden` 并中止。
- `DuplicateMitigation(2*time.Second)`：用于文章创建、评论创建，按用户和请求路径建立短时防重复 Key。
- `ViewHistoryMiddleware`：在可选文章详情路由上解析文章 ID，异步发送浏览历史 Kafka 消息，然后继续执行详情 Handler。

### 3.2 公开路由

| Method | Path | Handler | 请求参数/DTO | 成功响应 |
| --- | --- | --- | --- | --- |
| GET | `/article/list` | `ArticleHandler.GetPublishedList` | Query：`last_id`、`page`、`page_size`、`is_desc`；`GetPublishListRequest` | `ArticleListResponse`：列表、游标、总数、页码、页大小 |
| GET | `/article/hot-rank` | `ArticleHandler.GetHotArticleRank` | 无 | `HotRankResponse` |
| POST | `/user/register` | `UserAuthHandler.Register` | JSON：`nickname`、`phone`、`password`；注册字段必填，密码最少 6 位 | 统一响应，消息“注册成功” |
| POST | `/user/login` | `UserAuthHandler.Login` | JSON：`phone` 或 `nickname`、`password`、`remember_me`、`device` | `LoginResponse`：`access_token` |
| GET | `/user/profile` | `UserHandler.GetPublicProfile` | Query：`user_id` | `UserProfileResponse`：ID、昵称、头像 |
| GET | `/comment/list/roots` | `CommentHandler.ListRoots` | Query：文章 ID、分页/游标参数；见 `comment.GetRootCommentListRequest` | `RootCommentListResponse` |
| GET | `/comment/list/replies` | `CommentHandler.ListReplies` | Query：根评论 ID、分页/游标参数；见 `comment.GetReplyListRequest` | `ReplyListResponse` |

公开文章详情不在此组，而位于 `/optional`，因为它同时支持游客和登录用户两种行为。

### 3.3 登录用户路由

这些路由统一挂载在 `/auth` 下，并先执行 `MustAuth`。

| Method | Path | Handler | 请求参数/DTO | 成功响应 |
| --- | --- | --- | --- | --- |
| GET | `/auth/my/profile` | `UserHandler.GetMyProfile` | 无 | `MyProfileResponse` |
| POST | `/auth/my/profile/update` | `UserHandler.UpdateProfile` | JSON：`nickname`、`avatar`；`UpdateProfileRequest` | 空 Data，消息“个人资料修改成功” |
| POST | `/auth/my/password/verify` | `UserHandler.VerifyOldPassword` | JSON：`old_password` | `change_token` |
| POST | `/auth/my/password/change` | `UserHandler.ChangePassword` | JSON：`change_token`、`new_password` | 空 Data，消息“密码修改成功” |
| POST | `/auth/my/account/update` | `UserHandler.UpdateAccount` | JSON：`phone` | 空 Data，消息“电话修改成功” |
| POST | `/auth/my/avatar/upload-url` | `UserHandler.GetAvatarUploadURL` | JSON：`file_ext` | `avatar_url` |
| POST | `/auth/my/avatar/confirm` | `UserHandler.ConfirmAvatar` | JSON：`object_key` | 空 Data，消息“头像更新成功” |
| POST | `/auth/my/logout` | `UserHandler.Logout` | 从请求头读取 Bearer Token | 空 Data，消息“退出成功” |
| POST | `/auth/comment/create` | `CommentHandler.Create` | JSON：文章 ID、根评论 ID、回复用户 ID、正文；挂载 2 秒防重复 | `CreateCommentResponse`：ID、创建时间 |
| POST | `/auth/comment/delete` | `CommentHandler.DeleteMyComment` | JSON：评论 ID | 空 Data，消息“评论已成功删除” |
| POST | `/auth/article/like` | `LikeHandler.ArticleLikeHandler` | JSON：`article_id` | 空 Data，消息“点赞成功” |
| POST | `/auth/article/cancel_like` | `LikeHandler.ArticleCancelLikeHandler` | JSON：`article_id` | 空 Data，消息“取消点赞成功” |
| POST | `/auth/comment/like` | `LikeHandler.CommentLikeHandler` | JSON：`comment_id` | 空 Data，消息“点赞评论成功” |
| POST | `/auth/comment/cancel_like` | `LikeHandler.CommentCancelLikeHandler` | JSON：`comment_id` | 空 Data，消息“取消点赞评论成功” |
| GET | `/auth/ntf/unread-count` | `NotificationHandler.GetUnreadCount` | 无 | Data 为未读数量，消息“获取未读消息数量成功” |
| GET | `/auth/ntf/list` | `NotificationHandler.GetNotificationList` | Query：`page`、`page_size`；默认值由 Notification Service 处理 | `NotificationListResponse` |
| POST | `/auth/ntf/clear-unread` | `NotificationHandler.ClearUnread` | 无 | 空 Data，消息“消除未读消息成功” |

### 3.4 管理员路由

这些路由统一挂载在 `/admin` 下，并按顺序执行 `MustAuth`、`AdminCheckMiddleware`。

| Method | Path | Handler | 请求参数/DTO | 成功响应 |
| --- | --- | --- | --- | --- |
| POST | `/admin/article/create` | `ArticleHandler.CreateArticle` | JSON：`title`、`content`、`tags`、`status`；挂载 2 秒防重复 | 空 Data，消息“文章创建成功” |
| POST | `/admin/article/update` | `ArticleHandler.UpdateArticle` | JSON：`id`、`title`、`content`、`tags`、`status` | 空 Data，消息“文章更新成功” |
| POST | `/admin/article/delete` | `ArticleHandler.DeleteArticle` | JSON：`id` | 空 Data，消息“文章删除成功” |
| POST | `/admin/article/publish` | `ArticleHandler.PublishArticle` | JSON：`id` | 空 Data，消息“文章发表成功” |
| GET | `/admin/article/list` | `ArticleHandler.GetAdminList` | Query：`status`、`last_id`、`page`、`page_size`、`is_desc` | `AdminListResponse` |
| GET | `/admin/article/me/detail` | `ArticleHandler.GetArticleDetailForMe` | Query：`id` | `ArticleDetailResponse`，消息“查询成功” |
| GET | `/admin/article/trash/list` | `ArticleHandler.GetTrashList` | Query：分页/游标参数 | `AdminListResponse` |
| POST | `/admin/article/trash/recover` | `ArticleHandler.RecoverArticle` | JSON：`id` | 空 Data，消息“恢复文章成功” |
| POST | `/admin/article/trash/clear` | `ArticleHandler.ClearArticle` | JSON：`id` | 空 Data，消息“删除文章成功” |
| POST | `/admin/article/image/upload-url` | `ArticleHandler.GetImageUploadURL` | JSON：`file_ext` | `image_id`、`upload_url`、`url`，消息“获取成功” |
| POST | `/admin/comment/delete` | `CommentHandler.DeleteAdminComment` | JSON：评论 ID | 空 Data，消息“管理员已成功处理违规评论” |

### 3.5 可选认证路由

| Method | Path | Handler/中间件 | 请求参数/DTO | 成功响应/副作用 |
| --- | --- | --- | --- | --- |
| GET | `/optional/article/detail` | `OptionalAuth` → `ViewHistoryMiddleware` → `ArticleHandler.GetArticleDetail` | Query：`id` | `ArticleDetailResponse`，消息“查询成功”；异步发送浏览历史消息 |

`ArticleDetailResponse` 的关键字段为：`id`、`title`、`content`、`tags`、`status`、`author_nick`、`author_avatar`、`ip`、`created_time`、`updated_time`、`is_liked`、`like_count`、`images`。`content` 保留 `image://<image_id>`，`images` 提供当前公开 URL 映射。文章公开详情在游客场景以用户 ID 0 查询点赞状态；登录场景使用当前用户 ID。

### 3.6 统一响应和主要错误映射

成功响应结构保持：

```json
{
  "success": true,
  "code": 200,
  "message": "业务成功消息",
  "data": {}
}
```

失败响应由 `GlobalErrorMiddleware` 生成，结构保持：

```json
{
  "success": false,
  "code": 业务错误码,
  "message": "错误信息",
  "data": null
}
```

重构必须保留以下错误族及其业务码映射：

- 请求格式/认证：`ErrInvalidRequestBody`、`ErrAuthorizationRequired`、`ErrInvalidAuthorizationHeader`、`ErrTokenEmpty`、`ErrDuplicateSubmission`、`ErrForbidden`。
- 参数：`ErrParameter`、`ErrRegisterInputEmpty`、`ErrLoginInputEmpty`、`ErrRoleInvalid`、`ErrPasswordTooShort`、`ErrArticleTitleEmpty`、`ErrArticleContentEmpty`、`ErrArticleIDInvalid`、`ErrArticleStatusInvalid`。
- User：`ErrUserExists`、`ErrUserNotFound`、`ErrPasswordFailed`、`ErrPasswordChangeToken`、`ErrNickNameNotFound`、`ErrPhoneAlreadyExists`。
- Token：`ErrTokenInvalid`、`ErrTokenExpired`、`ErrTokenSignature`、`ErrTokenIssuer`、`ErrTokenRevoked`。
- Article：`ErrArticleNotFound`、`ErrArticleDeleted`、`ErrArticlePermissionDenied`、`ErrArticleStatusError`。
- Comment：`ErrCommentNotFound`、`ErrCommentDeleted`、`ErrCommentRootDeleted`、`ErrCommentPermission`。
- Redis/Kafka：现有锁、客户端、序列化、Topic、消费和关闭错误均保持原错误文本及映射。

### 3.7 HTTP 迁移检查点

- 路由路径、HTTP Method、分组前缀和 Handler 语义必须逐项对照本节。
- 请求字段名、绑定来源（JSON/Form/Query）、binding 规则和默认值必须逐项对照 `internal/dto/*/request.go`。
- 响应字段名、时间戳单位、分页字段和成功消息必须逐项对照 `internal/dto/*/response.go` 与 Handler。
- 登录、游客、普通用户和管理员场景必须分别验证。
- Handler 不应把基线中原本由中间件或 Service 处理的规则提前、延后或重复执行。

## 4. 对外 gRPC 契约清单

### 4.1 Service 与 Method

| Service | RPC Method | Request | Response | 入口校验 |
| --- | --- | --- | --- | --- |
| `blogopen.v1.UserService` | `GetUserBasicInfo` | `user_id` | `user_id`、`nickname`、`avatar`、`last_login_time`、`last_login_ip` | `user_id > 0` |
| `blogopen.v1.UserService` | `GetPublicUserInfo` | `user_id` | `id`、`avatar`、`nickname` | `user_id > 0` |
| `blogopen.v1.ArticleService` | `GetAvailableList` | `page`、`page_size`、`is_desc` | `items`、`total`、`page`、`page_size` | `page > 0`；`1 <= page_size <= 100` |
| `blogopen.v1.CommentService` | `GetCommentStats` | `comment_id` | `comment_id`、`hot_value`、`like_count` | `comment_id > 0` |

Protobuf 源文件位于 `proto/blogopen/v1/*.proto`，`gen/` 下生成文件只作为编译产物，重构不得直接编辑。

### 4.2 Interceptor 顺序与认证

Server 使用一元拦截器链：

```text
LoggingInterceptor（外层）
    ↓
AuthInterceptor
    ├── metadata 存在 x-access-key-id → HmacAuthInterceptor
    └── 否则 → JwtInterceptor
        ↓
Service Handler
```

- Logging 负责 Trace ID、方法名、请求/响应摘要、耗时、身份和错误日志。
- JWT 认证从 metadata 的 `authorization` 读取 `Bearer <token>`，使用 OpenJWT 校验，并注入内部身份。
- HMAC 认证读取 `x-access-key-id`、`x-signature`、`x-timestamp`、`x-nonce`，校验时间窗口、Redis nonce、防重放、请求体哈希和签名，并注入合作方身份。
- gRPC 错误映射由 `GRPCError` 完成：资源不存在映射 `NotFound`，权限映射 `PermissionDenied`，Token 映射 `Unauthenticated`，参数映射 `InvalidArgument`，未知错误记录日志后映射 `Internal`。

### 4.3 gRPC 迁移检查点

- 不改变 Service、RPC Method、字段编号、字段类型和 Go package 对外名称。
- 不改变 Logging → Auth 的拦截器顺序。
- 不把 HTTP Session Token 校验错误地替换为 gRPC 内部 JWT 校验，二者是不同认证链路。
- 不改变三方 HMAC 的签名原文组成：`access_key_id + method_name + timestamp + nonce + request_body_hash`。

## 5. Kafka 契约与处理链路

### 5.1 Topic 与 Consumer Group

| 配置 Key | Topic | Consumer Group | 当前用途 | Producer Key |
| --- | --- | --- | --- | --- |
| `notification` | `notification` | `blog_notification_consumer` | 文章点赞通知消费 | 无，轮询分区 |
| `view_history` | `view_history` | `blog_view_consumer` | 浏览历史消费 | `user_id` 字符串，Hash 分区 |
| dead letter | `dead_letter` | `blog_dead_letter_consumer` | 失败消息目标配置；当前 Consumer 的 `sendToDeadLetter` 只记录日志 | 由 Platform 配置决定 |

Topic 的批量、拉取、重试和提交参数来自现有配置；重构不得修改配置项语义或默认值。

### 5.2 消息 JSON

文章点赞通知消息：

```json
{
  "notify_type": 1,
  "sender_id": 123,
  "target_id": 456,
  "created_time": "2026-08-21T00:00:00Z"
}
```

浏览历史消息：

```json
{
  "article_id": 456,
  "user_id": 123,
  "created_time": "2026-08-21T00:00:00Z"
}
```

时间字段为 `time.Time` JSON 表示。当前消息没有 `event_id`、`version` 或 Envelope；重构不得自行新增这些字段。

### 5.3 Producer 行为

- 浏览历史由 `/optional/article/detail` 的 `ViewHistoryMiddleware` 异步触发，构造消息后同步等待 Kafka Producer ACK；发送失败记录日志，不影响主 HTTP 响应。
- 文章点赞数据库事务成功、Redis Set 更新后，调用异步通知 Producer；异步 Producer 使用独立 5 秒超时 context，发送失败只记录日志，不回滚已经完成的点赞。
- `notification` 使用 RoundRobin；`view_history` 使用 Hash，保证同一用户消息分区稳定。
- Producer 使用同步 `WriteMessages`，消息 JSON 由 `encoding/json` 序列化。

### 5.4 Consumer 行为

`kafka-consume` 注册两个 Handler：

```text
notification → 解析 NotificationMsg → 查询文章 → 查询发送方用户 → 写入通知
view_history → 解析 ViewHistoryMsg → 登录用户写浏览历史 → 原子增加文章 view_count
```

通用 Consumer 行为：

1. 按 topic 配置批量拉取消息；
2. 批次内按拉取顺序逐条处理；
3. 每条消息失败时按配置重试；
4. 重试最终失败时记录死信日志并继续处理批次其他消息；
5. 按 partition 记录最新 offset；
6. 无论业务处理成功或失败，当前实现都会尝试提交批次 offset；
7. offset 提交失败按配置重试，最终失败时允许消息重新消费；
8. 关闭时取消消费 context、等待 goroutine、关闭 Reader。

本基线没有业务级消费幂等，不能在重构中新增。

## 6. 数据、缓存、对象存储和 Cron

### 6.1 MySQL 表与写入所有权

| 表 | 当前主要写入方 | 重构目标上下文 | 关键说明 |
| --- | --- | --- | --- |
| `users` | `UserService`、`UserAuthService` | User | 手机号和昵称唯一；status 1 正常、2 删除/禁用 |
| `articles` | `ArticleService`、ArticleLikeService、CommentService、ArticleViewHistoryService | Article | 文章状态、浏览量、点赞数、评论数均为兼容字段 |
| `comments` | `CommentService` | Comment | 主评论/回复、软删除、根评论回复计数 |
| `article_likes` | `ArticleLikeService` | Like | `(user_id, article_id)` 唯一；status 1 点赞、2 取消 |
| `comment_likes` | `CommentLikeService` | Like | `(user_id, comment_id)` 唯一；status 1 点赞、2 取消 |
| `article_view_histories` | `ArticleViewHistoryService` | Article | 仅登录用户写入；同时增加文章浏览量 |

当前基线中 `ArticleLikeService` 会在点赞关系事务中更新 `articles.like_count`，`CommentService` 会在评论事务中更新 `articles.comment_count`，`ArticleViewHistoryService` 会直接原子增加 `articles.view_count`。这些是现有跨上下文副作用，不得在迁移时改成异步最终一致或拆成无事务的两个写操作。

### 6.2 MongoDB

- 数据库和集合：`blog.notifications`。
- 文档主键：MongoDB `ObjectID`，HTTP 响应转为 Hex 字符串。
- 关键字段：`receiver_id`、`sender`、`type`、`is_read`、`content`、`created_time`。
- 当前真正接线的通知类型：文章点赞通知 `type = 1`，内容包含 `article_id`、`article_title`。
- 当前已读操作：按接收者批量将 `is_read=false` 更新为 `true`；未读统计按接收者和 `is_read=false` 计数；通知列表按 `created_time` 倒序分页。

### 6.3 Redis Key 与缓存行为

以下 Key 是代码中声明或运行链路使用的兼容标识，重构不得重命名：

| Key | 类型/用途 | 所属目标上下文 |
| --- | --- | --- |
| `like:article:<article_id>` | Set，记录文章点赞用户 | Like |
| `lock:like:article:<article_id>` | Set 冷启动重建锁 | Like |
| `like:comment:<comment_id>` | Set，记录评论点赞用户 | Like |
| `lock:like:comment:<comment_id>` | Set 冷启动重建锁 | Like |
| `like:comment:count:<comment_id>` | 评论点赞数量兼容兜底读取 Key，代码注释标记待删除 | Comment/Like 交界，迁移前不得擅自删除 |
| `rank:article:hot` | ZSet，文章热榜 | Article |
| `lock:init:rank:article:hot` | 热榜初始化锁常量 | Article |
| `article:info:<article_id>` | 文章统计 Hash 前缀常量 | Article |
| `lock:init:article:info:<article_id>` | 文章信息 Hash 初始化锁常量 | Article |
| `hmac:nonce:<access_key_id>:<nonce>` | HMAC 防重放 nonce Key | Platform Security |
| `auth:token:<token>` | 登录 Token → Session | User/Platform Security | 默认 7 天或记住我 30 天过期 |
| `auth:user-tokens:<user_id>` | 用户 Token Set | User/Platform Security | 用于登出和修改密码后的会话清理 |
| `user:password-change:<token>` | 一次性密码修改凭证 | User | 10 分钟 TTL，消费后删除 |

点赞 Set 冷启动时先查 Redis，Set 不存在则尝试获取 3 秒锁；持锁时从数据库批量读取点赞用户并写入 Set，空集合写入占位成员 `0`，成功后设置 7 天过期；Redis 失败时回退数据库查询。点赞成功/取消成功后分别执行 `SAdd`/`SRem`，缓存失败只记录日志。

热榜初始化从文章表读取前 100 条，按 `view_count + 2*like_count + comment_count` 计算分数，使用 Redis Transaction Pipeline 清空并重写 ZSet；启动时初始化一次，Cron 每小时执行一次。

### 6.4 MinIO 对象路径

- 文章图片通过 `POST /admin/article/image/upload-url` 按单张实时申请凭证，不要求预先创建文章。
- 新图片对象路径为 `article/img/<year>/<month>/<uuid>.<ext>`，预签名 PUT URL 有效期 10 分钟。
- 图片上传时创建 `article_id = NULL` 的图片记录；PUT 成功后前端在正文保存 `image://<image_id>`。
- 创建和更新文章时在同一数据库事务中同步图片归属，移除引用时只解绑为 `NULL`，不立即删除对象。
- 文章详情保留原始正文，并返回正文引用且属于当前文章的图片 ID—URL 映射。
- 文章软删除和恢复保留图片关系；硬删除按 `article_id` 清理对象和图片记录，清理失败时保留文章以便重试。
- 未绑定图片定时清理不属于当前实现范围。
- 用户头像上传凭证生成：`avatar/<user_id>/<uuid>.<ext>`，预签名 PUT URL 有效期 10 分钟。
- 确认头像时只接受当前用户目录前缀 `avatar/<user_id>/`，数据库保存对象 Key，响应返回公开域名拼接后的完整 URL。

### 6.5 Cron

- Job 名称：`rank_calibrate_daily`。
- 当前表达式：`0 0 * * * *`，实际按秒级 Cron 表达式每小时整点执行。
- 单次执行超时：5 分钟。
- Cron Manager 使用 `SkipIfStillRunning` 和 `Recover`，启动时注册并启动，退出时 Stop 并等待完成。
- 启动阶段还会先以 3 秒超时执行一次热榜重建。

## 7. 关键副作用与事务顺序

### 7.1 文章点赞

```text
1. 查询 Redis 点赞 Set；Set 不存在时按锁/数据库回源判断
2. 查询数据库中是否已有该用户记录
3. 本地 MySQL 事务
   3.1 已有记录则更新 article_likes.status
   3.2 无记录则插入 article_likes
   3.3 更新 articles.like_count ±1
4. 事务提交
5. Redis 对 like:article:<article_id> 执行 SAdd 或 SRem
6. 点赞成功时异步发送 notification Kafka 消息
```

重复点赞或重复取消保持幂等返回，不新增目标存在性查询。排行榜即时更新代码当前被注释，不得在重构中重新启用。

### 7.2 评论点赞

```text
1. 查询 Redis 评论点赞 Set；Set 不存在时按锁/数据库回源判断
2. 查询 comment_likes 是否已有该用户记录
3. 本地 MySQL 事务
   3.1 更新或插入 comment_likes.status
   3.2 更新 comments.like_count ±1
4. 事务提交
5. Redis 对 like:comment:<comment_id> 执行 SAdd 或 SRem
```

当前评论点赞没有通知 Kafka 发布，也没有文章计数更新。

### 7.3 创建评论

```text
1. 校验文章、根评论、回复目标及请求内容
2. 组装评论模型
3. 本地 MySQL 事务
   3.1 插入 comments
   3.2 主评论时增加文章 comment_count；回复时增加根评论 comment_count 和文章 comment_count
4. 事务提交
5. 组装 CreateCommentResponse 返回评论 ID 和创建时间
```

当前未接线评论创建事件或评论通知消息；不得在迁移中补齐。

### 7.4 删除评论

主评论：

```text
1. 查询评论并校验未删除及操作者权限
2. 本地 MySQL 事务
   2.1 批量软删除所有子评论并获取有效子评论数
   2.2 软删除主评论
   2.3 文章 comment_count 一次扣减自身和子评论总数
3. 提交后返回成功
```

子评论：

```text
1. 查询评论并校验未删除及操作者权限
2. 本地 MySQL 事务
   2.1 软删除当前子评论
   2.2 根评论 comment_count -1
   2.3 文章 comment_count -1
3. 提交后返回成功
```

并发下更新影响行数为 0 时按当前实现直接提交，不新增锁或重试规则。

### 7.5 浏览历史

```text
HTTP 详情请求成功链路前：
1. OptionalAuth 获取可选用户身份
2. ViewHistoryMiddleware 解析 article_id
3. 新 goroutine 使用独立 5 秒 context 发送 view_history Kafka 消息
4. Handler 查询并返回文章详情

Kafka 消费链路：
1. 解析 ViewHistoryMsg
2. user_id > 0 时插入 article_view_histories；写入失败仅记录日志
3. 原子增加 articles.view_count；失败返回错误触发 Consumer 重试
```

当前浏览历史消息发送不阻塞主请求；消费写历史失败不阻止阅读量增加尝试。

### 7.6 文章点赞通知

```text
点赞事务和 Redis 更新完成
    ↓
异步发送 NotificationMsg 到 notification Topic
    ↓
Consumer 解析消息
    ↓
按 target_id 查询文章
    ↓
按 sender_id 查询用户
    ↓
若 sender_id == receiver_id 则不创建通知
    ↓
写入 MongoDB notifications
```

通知发送失败不回滚已成功的点赞；当前没有业务幂等、Outbox 或重放去重。

## 8. 当前测试覆盖与替代映射

### 8.1 可保留或迁移的测试

| 现有测试 | 归属上下文/平台 | 当前覆盖 | 迁移处理 |
| --- | --- | --- | --- |
| `internal/auth/hmac_test.go` | Platform Security | HMAC 签名、method/body/nonce 篡改失败 | 迁移到 Security Adapter/纯函数测试 |
| `internal/auth/password_test.go` | User/Platform Security | PBKDF2 密码生成与校验 | 保留语义测试，技术实现迁移后替换包路径 |
| `internal/common/response_test.go` | Platform HTTP | 统一响应结构 | 迁移到 HTTP response/error mapper 测试 |
| `internal/grpc/interceptor/hmac_auth_test.go` | Platform gRPC | HMAC 拦截器认证和 nonce | 迁移到 gRPC Security Interfaces 测试 |
| `internal/handler/article_handler_test.go` | Article HTTP | 部分 Handler 流程 | 迁移为 Article HTTP 契约/Handler 测试 |
| `internal/handler/user_auth_handler_test.go` | User HTTP | 注册、登录成功和错误路径 | 迁移为 User HTTP 契约测试 |
| `internal/handler/user_handler_test.go` | User HTTP | 个人资料、公开资料、资料/密码修改 | 迁移为 User HTTP/Application 测试 |
| `internal/middleware/duplicate_test.go` | Platform HTTP | 防重复提交 | 迁移到 Duplicate Adapter/中间件测试 |
| `internal/repository/article_repository_test.go` | Article Infrastructure | 当前大部分为注释测试代码 | 重新建立 Article Repository 集成测试 |
| `internal/repository/user_repository_test.go` | User Infrastructure | 当前大部分为注释测试代码 | 重新建立 User Repository 集成测试 |
| `internal/service/article_service_test.go` | Article Application | 当前大部分为注释测试代码 | 重新建立文章用例测试 |
| `internal/service/user_auth_service_test.go` | User Application | 用户认证业务测试 | 迁移为 User Application 测试 |
| `internal/service/user_service_test.go` | User Application | 用户资料业务测试 | 迁移为 User Application 测试 |
| `pkg/util/ip/ip_test.go` | Platform IP | IP 区域解析 | 迁移后保留资源定位回归测试 |

### 8.2 当前没有有效测试覆盖的链路

以下基线能力在当前测试目录中没有成体系的测试，后续上下文迁移时需要新增，而不是假定已有测试保护：

- Comment 创建、回复、删除级联和评论统计；
- Like 文章/评论点赞、Redis 冷启动、锁降级和事务回滚；
- Notification MongoDB 列表、未读、清理和 Kafka Consumer；
- Kafka Producer/Consumer 批量重试、offset 提交和关闭；
- MinIO 文章图片关系同步与硬删除、头像确认；
- Cron 热榜初始化与周期重建；
- 完整 HTTP 路由注册和中间件顺序；
- 完整 gRPC descriptor、JWT/HMAC 双路认证契约。

### 8.3 已知基线测试环境问题

在当前环境使用独立 `GOCACHE` 运行 `go test ./...` 时，已观察到以下既有问题：

1. `internal/handler` 的登录测试触发未注册的 `not_only_number` Validator，发生 panic；
2. `pkg/util/ip` 测试依赖相对路径 `../resource/ip2region.xdb`，从包测试目录运行时找不到资源。

这些问题不作为本次基线行为变更；后续测试迁移时需要通过等价测试或测试夹具修复解决，不能直接删除相关行为覆盖。

## 9. 重构前验收清单

在进入下一个大 Task 前，必须确认：

- [x] HTTP 路由、Method、分组、中间件顺序和主要 DTO/响应已记录。
- [x] gRPC Service、Method、字段、拦截器顺序和错误映射已记录。
- [x] Kafka Topic、Group、消息字段、生产/消费顺序、重试和 offset 行为已记录。
- [x] 数据表、集合、Redis Key、MinIO 路径、Cron 和写入所有权已记录。
- [x] 点赞、评论、浏览历史和通知的关键副作用顺序及事务边界已记录。
- [x] 旧测试到上下文和外部契约的映射已记录。
- [x] 已标记当前测试空白和基线测试环境问题。

该清单只固定基线，不代表已经开始迁移任何业务上下文。
