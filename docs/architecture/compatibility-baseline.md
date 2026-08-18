# 兼容性基线

本文档固化当前博客单体在 HTTP、鉴权、统一响应、错误码、开放 gRPC 与 Kafka 上的外部行为，作为阶段一 DDD 重构的兼容性清单。阶段一任何代码改动不得改变本文记录的行为。

## 1. HTTP 契约

### 1.1 统一入口

- 进程：`blog server`，默认监听 `:8080`。
- 全局中间件顺序：`LoggerMiddleware` -> `GlobalErrorMiddleware`。
- 请求头：`Authorization: Bearer <token>`，可选 `X-Trace-ID`。
- 响应头：`X-Trace-ID` 由日志中间件写入。
- 业务接口成功与失败均返回 HTTP 200，通过响应体中的 `success` 和 `code` 区分。
- 未匹配路由仍由 Gin 返回默认 404，不属于业务接口范围。

### 1.2 路由与鉴权分组

| 方法 | 路径 | 鉴权分组 | Handler | 备注 |
| --- | --- | --- | --- | --- |
| POST | `/user/register` | public | `UserAuthHandler.Register` | 注册 |
| POST | `/user/login` | public | `UserAuthHandler.Login` | 登录 |
| GET | `/user/profile` | public | `UserHandler.GetPublicProfile` | 他人主页 |
| GET | `/article/list` | public | `ArticleHandler.GetPublishedList` | 已发表文章列表 |
| GET | `/article/hot-rank` | public | `ArticleHandler.GetHotArticleRank` | 热榜 |
| GET | `/comment/list/roots` | public | `CommentHandler.ListRoots` | 主评论列表 |
| GET | `/comment/list/replies` | public | `CommentHandler.ListReplies` | 楼中楼回复 |
| GET | `/optional/article/detail` | optional | `ArticleHandler.GetArticleDetail` | 文章详情，前置 `ViewHistoryMiddleware` |
| GET | `/auth/my/profile` | private | `UserHandler.GetMyProfile` | 个人主页 |
| POST | `/auth/my/profile/update` | private | `UserHandler.UpdateProfile` | 更新昵称/头像 |
| POST | `/auth/my/password/verify` | private | `UserHandler.VerifyOldPassword` | 旧密码验证并签发一次性改密凭证 |
| POST | `/auth/my/password/change` | private | `UserHandler.ChangePassword` | 使用一次性凭证改密 |
| POST | `/auth/my/account/update` | private | `UserHandler.UpdateAccount` | 更新手机号 |
| POST | `/auth/my/avatar/upload-url` | private | `UserHandler.GetAvatarUploadURL` | 头像上传凭证 |
| POST | `/auth/my/avatar/confirm` | private | `UserHandler.ConfirmAvatar` | 确认头像上传完成 |
| POST | `/auth/my/logout` | private | `UserHandler.Logout` | 退出登录并失效当前会话 |
| POST | `/auth/comment/create` | private | `CommentHandler.Create` | 创建评论/回复，前置防重复 |
| POST | `/auth/comment/delete` | private | `CommentHandler.DeleteMyComment` | 用户删除自己的评论 |
| POST | `/auth/article/like` | private | `LikeHandler.ArticleLikeHandler` | 文章点赞 |
| POST | `/auth/article/cancel_like` | private | `LikeHandler.ArticleCancelLikeHandler` | 取消文章点赞 |
| POST | `/auth/comment/like` | private | `LikeHandler.CommentLikeHandler` | 评论点赞 |
| POST | `/auth/comment/cancel_like` | private | `LikeHandler.CommentCancelLikeHandler` | 取消评论点赞 |
| GET | `/auth/ntf/unread-count` | private | `NotificationHandler.GetUnreadCount` | 未读通知数 |
| GET | `/auth/ntf/list` | private | `NotificationHandler.GetNotificationList` | 通知列表 |
| POST | `/auth/ntf/clear-unread` | private | `NotificationHandler.ClearUnread` | 清空未读 |
| POST | `/admin/article/create` | admin | `ArticleHandler.CreateArticle` | 创建文章，前置防重复 |
| POST | `/admin/article/update` | admin | `ArticleHandler.UpdateArticle` | 更新文章 |
| POST | `/admin/article/delete` | admin | `ArticleHandler.DeleteArticle` | 软删除文章 |
| POST | `/admin/article/publish` | admin | `ArticleHandler.PublishArticle` | 发布文章 |
| GET | `/admin/article/list` | admin | `ArticleHandler.GetAdminList` | 后台文章列表 |
| GET | `/admin/article/me/detail` | admin | `ArticleHandler.GetArticleDetailForMe` | 后台文章详情 |
| GET | `/admin/article/trash/list` | admin | `ArticleHandler.GetTrashList` | 垃圾箱列表 |
| POST | `/admin/article/trash/recover` | admin | `ArticleHandler.RecoverArticle` | 恢复文章 |
| POST | `/admin/article/trash/clear` | admin | `ArticleHandler.ClearArticle` | 彻底删除文章 |
| POST | `/admin/article/image/upload-url` | admin | `ArticleHandler.GetImageUploadURL` | 文章图片上传凭证 |
| POST | `/admin/comment/delete` | admin | `CommentHandler.DeleteAdminComment` | 管理员处理违规评论 |

### 1.3 鉴权语义

| 分组 | 前缀 | 中间件 | 行为 |
| --- | --- | --- | --- |
| public | 无 | 无 | 游客可直接访问 |
| optional | `/optional` | `OptionalAuth` | 有有效会话则注入用户，否则按游客继续执行 |
| private | `/auth` | `MustAuth` | 必须携带有效 `Authorization: Bearer <token>`；会话来自 Redis |
| admin | `/admin` | `MustAuth` + `AdminCheckMiddleware` | 必须登录且角色为管理员；非管理员返回 `1003` |

防重复中间件 `DuplicateMitigation` 作用于 `/auth/comment/create` 和 `/admin/article/create`，以 `user_id + 路由完整路径` 为内存级去重键，窗口为 2 秒，命中返回 `1004`。

### 1.4 统一响应结构

```json
{
  "success": true,
  "code": 200,
  "message": "获取成功",
  "data": {}
}
```

- 成功：`common.OK`，`success=true`，`code=200`。
- 失败：`common.Fail`，`success=false`，`code=业务错误码`，`data=null`。
- HTTP 状态码固定为 200。
- `message` 为错误文本或成功文案，属于现有对外契约。

### 1.5 错误码清单

| 错误码 | 含义 |
| --- | --- |
| 200 | 成功 |
| 1000 | 请求体 JSON 格式错误 |
| 1001 | 参数校验失败 |
| 1002 | 未登录/认证失败 |
| 1003 | 无权限 |
| 1004 | 重复提交 |
| 5000 | 系统异常 |
| 1100 | 用户已存在 |
| 1101 | 用户不存在 |
| 1102 | 密码错误 |
| 1103 | 用户被禁用 |
| 1104 | 昵称缺失 |
| 1105 | 手机号已被注册 |
| 1200 | Token 无效 |
| 1201 | Token 过期 |
| 1300 | 文章不存在 |
| 1301 | 文章被删除 |
| 1302 | 操作文章权限不足 |
| 1303 | 文章状态异常 |
| 1400 | 评论不存在 |
| 1401 | 评论已被删除 |
| 1402 | 主楼评论已被删除，无法回复 |
| 1403 | 操作评论权限不足 |
| 1500 | 解锁失败 |
| 1501 | 加锁失败 |
| 1502 | 锁过期 |
| 1600 | Kafka 初始化失败 |
| 1601 | Kafka 消息发送失败 |
| 1602 | Kafka 消息序列化失败 |
| 1603 | Kafka Topic 未配置 |
| 1604 | Kafka 消息消费失败 |
| 1605 | Kafka 关闭失败 |
| 1606 | Kafka 无有效消费者 |
| 1607 | Kafka 消费者已运行 |
| 1608 | Kafka 客户端已关闭 |
| 1609 | Kafka 预热连接失败 |

## 2. 开放 gRPC 契约

proto 源文件位于 `proto/blogopen/v1/`，生成代码位于 `gen/`。

| Service | Method | 请求 | 响应 | 入参约束 |
| --- | --- | --- | --- | --- |
| `UserService` | `GetUserBasicInfo` | `GetUserBasicInfoRequest{user_id}` | `GetUserBasicInfoResponse{user_id,nickname,avatar,last_login_time,last_login_ip}` | `user_id > 0` |
| `UserService` | `GetPublicUserInfo` | `GetUserInfoRequest{user_id}` | `GetUserInfoResponse{id,avatar,nickname}` | `user_id > 0` |
| `ArticleService` | `GetAvailableList` | `GetExternalListRequest{page,page_size,is_desc}` | `ExternalListResponse{items,total,page,page_size}` | `page > 0`，`1 <= page_size <= 100` |
| `CommentService` | `GetCommentStats` | `GetCommentStatsRequest{comment_id}` | `GetCommentStatsResponse{comment_id,hot_value,like_count}` | `comment_id > 0` |

认证通过 gRPC metadata 区分：

- 内部二方：`authorization: Bearer <open-jwt>`，校验 `service_id`、`team_id`，签名算法锁定 HS256，签发者固定为 `blog-open`，有效期 24 小时。
- 三方合作方：`x-access-key-id`、`x-signature`、`x-timestamp`、`x-nonce`，HMAC-SHA256，时间戳窗口 60 秒，nonce 在 Redis 中 60 秒内去重。
- 链路：`x-trace-id` 由日志拦截器读取或生成。

错误映射：

| 业务错误 | gRPC code |
| --- | --- |
| 用户/文章/评论不存在 | `NotFound` |
| 权限不足 | `PermissionDenied` |
| Token 无效/过期 | `Unauthenticated` |
| 参数错误 | `InvalidArgument` |
| 其他 | `Internal` |

## 3. Kafka 契约

当前 Kafka 事件为 JSON 消息，暂不含事件 ID 与版本字段；后续阶段二扩展时必须采用兼容的追加式字段，不能改变现有消费者解析行为。

| topic key | topic 名 | consumer group | 消息类型 | 消费行为 |
| --- | --- | --- | --- | --- |
| `notification` | `notification` | `blog_notification_consumer` | `NotificationMsg{notify_type,sender_id,target_id,created_time}` | 创建点赞通知 |
| `view_history` | `view_history` | `blog_view_consumer` | `ViewHistoryMsg{article_id,user_id,created_time}` | 写入浏览历史并维护统计 |

死信 topic 为 `dead_letter`，consumer group 为 `blog_dead_letter_consumer`。消费者按 topic 配置重试 `max_retries` 次，超过上限进入死信处理。

## 4. 兼容性不变量

1. 路由的方法、路径和鉴权分组不得改变。
2. 统一响应 JSON 字段 `success/code/message/data` 不得改变。
3. 业务错误码值不得改变。
4. 开放 gRPC 的 Service、方法、请求/响应字段和鉴权语义不得改变。
5. Kafka topic、consumer group 和现有 JSON 字段不得破坏性变更。
6. 阶段一不得新增独立微服务进程、数据库实例或部署单元。
