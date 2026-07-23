# Go 博客系统

一个基于 Go、Gin 和 Cobra 的博客后端服务。当前实现覆盖用户认证、文章发布管理、评论、点赞、浏览量、热度榜单和站内通知。

## 技术栈与分层

- **HTTP/API**：Gin；Cobra 提供 `server`、`migrate` 命令。
- **Handler**：解析 JSON/Query、调用 service、输出统一响应。
- **Service**：承载状态校验、权限、事务编排、缓存和异步任务。
- **Repository**：GORM + MySQL 访问用户、文章、评论、点赞和浏览历史；MongoDB 访问通知。
- **缓存/并发**：Redis 保存点赞 Set、文章热度 ZSet；使用 Redis 锁进行冷缓存初始化；内存窗口限制重复浏览和重复提交。
- **安全**：PBKDF2-SHA256（100000 次迭代）密码哈希；HS256 JWT，默认有效期 24 小时；IP 使用 ip2region 转换为地区展示。

启动时会初始化 MySQL、MongoDB、Redis 和 IP 数据文件，并启动热度榜单定时校准任务。全局中间件提供 Trace ID 日志和统一错误响应。

## 项目目录

```text
.
├── cmd/                    # Cobra 命令、依赖组装和服务启动
├── config/                 # YAML 配置读取
├── internal/
│   ├── auth/               # JWT 与密码哈希
│   ├── common/             # 响应、错误码、验证器、防重和浏览窗口
│   ├── cron/               # 热度榜定时校准
│   ├── dto/                # article/comment/like/notification/user 请求响应模型
│   ├── handler/            # Gin HTTP 处理器
│   ├── middleware/         # 认证、管理员校验、日志、异常、防重
│   ├── model/              # MySQL/MongoDB 持久化模型与状态枚举
│   ├── repository/         # MySQL/MongoDB 数据访问
│   ├── routes/             # 公开、登录、管理员、可选登录路由
│   └── service/            # 业务规则、事务、缓存和异步任务
├── pkg/                    # 数据库客户端、IP、Redis 锁和缓存工具
└── scripts/                # MySQL 建库建表脚本、MongoDB 初始化脚本
```

## 当前功能

### 用户

- 手机号或昵称登录，注册时校验昵称、手机号和密码（至少 6 位）。
- JWT 登录态，普通用户角色为 `1`，管理员为 `2`。
- 查询他人主页；登录后查询/修改自己的昵称、头像、密码和手机号。
- 注册与登录 IP、最后登录时间会保存；密码不会明文存储。

### 文章

- 创建、更新、发布、软删除（移入垃圾箱）、恢复和硬删除。
- 文章状态：`1` 已删除、`2` 草稿、`3` 已发表。
- 前台只展示已发表文章；后台按状态查看文章和垃圾箱。
- 文章详情返回作者信息、标签、地区化 IP、点赞状态和点赞数。
- 文章列表和后台列表支持 offset 分页或 `last_id` 游标分页，每页 10-20 条。
- 文章详情浏览会异步增加浏览量；登录用户同时写入浏览历史，同一用户/访客 IP 对同一文章 10 分钟内只计一次。
- 热度榜单返回前 10 条，热度计算为 `浏览量 + 2*点赞数 + 评论数`；点赞或定时任务会刷新 Redis 榜单。

### 评论与点赞

- 发表评论：支持文章主评论和楼中楼回复，可指定被回复用户。
- 公开查询主评论、子评论，支持按作者过滤主评论以及 offset/游标分页。
- 用户只能软删除自己的评论；管理员可处理任意评论，删除主评论会连带处理子评论。
- 登录用户可点赞/取消点赞文章和评论；数据库事务更新状态及计数，Redis Set 加速查询。
- 创建文章和评论接口带 2 秒重复提交保护。

### 通知

- 文章被点赞后异步向作者写入 MongoDB 通知（自己操作自己不会产生通知）。
- 登录用户可查询通知列表、获取未读数量并一键清除未读状态。
- 模型预留了评论点赞、评论文章、回复评论类型，但当前 service 只实际发送文章点赞通知。

## API 路由

所有成功或失败结果均为统一结构：

```json
{"success": true, "code": 200, "message": "...", "data": {}}
```

需要登录的请求携带 `Authorization: Bearer <access_token>`。管理员接口除 JWT 外还要求 JWT 中的 `role=2`。

| 范围 | 方法 | 路径 | 说明 |
| --- | --- | --- | --- |
| 公开 | POST | `/user/register` | 注册 |
| 公开 | POST | `/user/login` | 登录并返回 JWT |
| 公开 | GET | `/user/profile?user_id=` | 他人主页 |
| 公开 | GET | `/article/list` | 已发表文章列表 |
| 公开 | GET | `/article/hot-rank` | 热度榜前 10 |
| 公开 | GET | `/comment/list/roots` | 主评论列表 |
| 公开 | GET | `/comment/list/replies` | 子评论列表 |
| 可选登录 | GET | `/optional/article/detail?id=` | 已发表文章详情；登录时返回 `is_liked` |
| 登录 | GET | `/auth/my/profile` | 我的主页 |
| 登录 | POST | `/auth/my/profile/update` | 更新昵称/头像 |
| 登录 | POST | `/auth/my/password/update` | 更新密码 |
| 登录 | POST | `/auth/my/account/update` | 更新手机号 |
| 登录 | POST | `/auth/comment/create` | 创建评论 |
| 登录 | POST | `/auth/comment/delete` | 删除自己的评论 |
| 登录 | POST | `/auth/article/like` / `/auth/article/cancel_like` | 文章点赞/取消 |
| 登录 | POST | `/auth/comment/like` / `/auth/comment/cancel_like` | 评论点赞/取消 |
| 登录 | GET | `/auth/ntf/unread-count` | 获取未读通知数 |
| 登录 | GET | `/auth/ntf/list` | 获取通知列表 |
| 登录 | POST | `/auth/ntf/clear-unread` | 全部标记为已读 |
| 管理员 | POST | `/admin/article/create` | 创建草稿或文章 |
| 管理员 | POST | `/admin/article/update` | 更新文章 |
| 管理员 | POST | `/admin/article/delete` | 移入垃圾箱 |
| 管理员 | POST | `/admin/article/publish` | 发布文章 |
| 管理员 | GET | `/admin/article/list` | 按状态查看文章 |
| 管理员 | GET | `/admin/article/me/detail` | 查看后台文章详情 |
| 管理员 | GET/POST | `/admin/article/trash/*` | 垃圾箱列表、恢复、硬删除 |
| 管理员 | POST | `/admin/comment/delete` | 强制删除评论 |

## 配置与运行

配置文件为 `config/config.yaml`，包含 MySQL、Redis、MongoDB 连接信息。请先按实际环境修改账号、密码、地址和端口，尤其不要把示例凭据用于生产环境。

```bash
# 初始化 blog 数据库及 scripts/mysql 下的业务表
go run . migrate

# 启动 HTTP 服务，默认监听 8080
go run . server
# 或指定端口
go run . server --port 9000
```

MongoDB 通知集合和索引可执行 `scripts/mongodb/notification.js` 初始化。服务启动依赖 `pkg/resource/ip2region.xdb`；MySQL 或 Redis 连接失败会阻止服务启动。当前启动流程没有显式处理 MongoDB 初始化错误，部署前应确保 MongoDB 可连接。

## 数据表与存储

- MySQL：`users`、`articles`、`comments`、`article_likes`、`comment_likes`、`article_view_histories`。
- MongoDB：`notifications` 集合，按接收者和已读状态建立索引。
- Redis：文章/评论点赞 Set、文章热度 ZSet，以及缓存初始化锁。

## 当前实现边界

- JWT 密钥目前写在 `internal/auth/jwt.go`，尚未接入配置中心或密钥轮换。
- 退出登录接口尚未注册路由，代码中的 `Logout` 仅为占位实现，也没有 JWT 黑名单。
- 通知列表 DTO 已定义多种通知类型，但目前只有“文章点赞”会产生并完整渲染通知。
- 文章/评论点赞关系缓存缺失时会在 Redis 锁保护下从 MySQL 重建；加锁失败时降级查询 MySQL。
- SQL 脚本中的历史注释状态值可能与当前 Go 枚举不同，运行行为以 `internal/model` 和 service 校验为准。

## 测试

项目包含 auth、middleware、handler、service 和 repository 测试，可运行：

```bash
go test ./...
```
