# Go 博客系统

一个基于 Go + Gin + Cobra 的博客后端服务，采用 Handler / Service / Repository 分层架构。除 HTTP 接口外，还提供面向二方与三方的 gRPC 开放接口，并使用 Kafka 承载通知、浏览历史等异步链路。

服务由同一个二进制的三个子命令组成：`server`（HTTP 服务）、`grpc`（开放 API 服务）、`kafka-consume`（异步消费者），另有 `migrate` 用于初始化数据库。

## 目录

- [功能总览](#功能总览)
- [角色与权限](#角色与权限)
- [技术栈](#技术栈)
- [架构与异步链路](#架构与异步链路)
- [接口一览](#接口一览)
- [项目结构](#项目结构)
- [快速开始](#快速开始)

## 功能总览

### 账号与认证

- 手机号或昵称登录，注册校验昵称、手机号唯一性与密码长度（≥ 6 位）。
- 密码使用 PBKDF2-SHA256 加盐哈希存储，不保存明文。
- 基于 Redis 的不透明 Token 会话：默认 7 天，勾选「记住我」延长至 30 天，支持记录设备标识、多端会话管理与主动登出（服务端可失效 Token）。
- 记录注册/最后登录 IP 与登录时间，IP 通过 ip2region 离线库转换为地区文案展示。
- 改密采用两步式：先校验旧密码换取一次性 `change_token`，再用该凭证提交新密码，改密后可清理其他会话。

### 个人主页与头像

- 查看他人公开主页，登录后查看与修改自己的昵称、头像、手机号。
- 头像走对象存储直传：服务端签发预签名上传地址，客户端上传完成后回调确认，服务端再落库最终访问地址。

### 文章

- 全生命周期管理：创建、编辑、发布、软删除（进垃圾箱）、恢复、彻底删除。
- 状态机：`1` 已删除 / `2` 草稿 / `3` 已发表；查询侧另支持 `-1` 全部（不含删除）与 `-2` 全部。
- 前台仅展示已发表文章，后台可按状态分页查看，并有独立垃圾箱列表。
- 列表同时支持 offset 分页与 `last_id` 游标分页，适配深翻页场景。
- 文章详情返回作者信息、标签、地区化 IP、点赞数与当前用户是否点赞。
- 正文图片直传：先获取预签名地址上传到临时目录，文章保存时将引用图片「转正」到正式目录，避免草稿废图污染。
- 浏览量与浏览历史异步统计：详情接口经中间件投递 Kafka 消息，同一用户/访客 IP 对同一文章在窗口期内只计一次。
- 热榜：热度 = `浏览量 + 2×点赞数 + 评论数`，Redis ZSet 维护 Top 10，点赞行为实时增量刷新，定时任务每小时全量校准，冷缓存重建用 Redis 分布式锁防止击穿。

### 评论

- 支持文章主评论与楼中楼回复，可指定被回复用户。
- 游客可查询主评论与子评论列表，支持按作者过滤与 offset / 游标分页。
- 用户可删除自己的评论；管理员可删除任意违规评论，删除主评论会级联处理其子评论。

### 点赞

- 登录用户可对文章与评论点赞 / 取消点赞。
- 数据库事务保证点赞状态与计数一致，Redis Set 缓存点赞关系加速「是否已点赞」查询。

### 消息通知

- 文章被点赞后经 Kafka 异步为作者生成通知，自赞不产生通知。
- 通知存储在 MongoDB，支持未读数量查询、通知列表查询与一键清空未读。
- 通知类型已定义点赞文章、点赞评论、评论文章、回复评论四类，当前实际投递的是点赞文章通知。

### 开放 API（gRPC）

- 用户：查询用户基本信息（二方）、查询用户公开信息（三方，仅返回 ID / 昵称 / 头像）。
- 文章：分页查询全部可用文章列表（不含已删除）。
- 评论：查询单条评论的点赞数与热度值。
- 统一认证拦截器按调用方身份自动分流：携带 `x-access-key-id` 的三方走 HMAC-SHA256 签名校验（时间戳 60s 窗口 + Redis nonce 防重放），二方走独立密钥的 JWT 校验。

### 工程能力

- 统一响应结构与错误码，全局异常兜底中间件，带 Trace ID 的请求日志。
- 创建文章、发表评论等写接口带 2 秒防重复提交保护。
- Kafka 消息支持批量消费、失败重试与死信队列（含死信消费者）。
- 提供 Dockerfile、docker-compose（开发/生产）与 Kubernetes 清单，服务与消费者独立部署。
- 关键模块（认证、密码、响应、防重、文章/用户 service 与 repository、handler）配有单元测试。

## 角色与权限

系统区分四类调用方身份：游客、普通用户（`role=1`）、管理员（`role=2`）、外部调用方（二方 / 三方）。

| 能力 | 游客 | 普通用户 | 管理员 |
| --- | :---: | :---: | :---: |
| 注册 / 登录 | ✅ | — | — |
| 浏览已发表文章列表与详情 | ✅ | ✅ | ✅ |
| 查看热度榜单 | ✅ | ✅ | ✅ |
| 查看主评论 / 子评论 | ✅ | ✅ | ✅ |
| 查看他人公开主页 | ✅ | ✅ | ✅ |
| 详情页展示「是否已点赞」 | ❌ | ✅ | ✅ |
| 写入浏览历史 | ❌（仅计浏览量） | ✅ | ✅ |
| 查看 / 修改个人资料、手机号、密码 | ❌ | ✅ | ✅ |
| 头像上传与确认、退出登录 | ❌ | ✅ | ✅ |
| 发表评论、删除自己的评论 | ❌ | ✅ | ✅ |
| 点赞 / 取消点赞文章与评论 | ❌ | ✅ | ✅ |
| 通知未读数、通知列表、清空未读 | ❌ | ✅ | ✅ |
| 创建 / 编辑 / 发布文章 | ❌ | ❌ | ✅ |
| 后台文章列表（按状态）与文章详情 | ❌ | ❌ | ✅ |
| 垃圾箱：软删除、恢复、彻底删除 | ❌ | ❌ | ✅ |
| 文章图片上传凭证 | ❌ | ❌ | ✅ |
| 删除任意用户评论（级联子评论） | ❌ | ❌ | ✅ |

外部调用方（gRPC，无 HTTP 权限）：

| 身份 | 认证方式 | 可用能力 |
| --- | --- | --- |
| 二方（内部服务） | 独立密钥签发的 JWT | 用户基本信息、可用文章列表、评论统计 |
| 三方（合作方） | AccessKey + HMAC-SHA256 签名 + 时间戳 + nonce | 仅用户公开信息（ID / 昵称 / 头像） |

权限实现方式：`MustAuth` 校验登录态（`/auth`、`/admin`），`AdminCheckMiddleware` 校验 `role=2`（`/admin`），`OptionalAuth` 用于登录与未登录返回内容不同的接口（`/optional`）。

## 技术栈

| 分类 | 选型 | 说明 |
| --- | --- | --- |
| 语言 | Go 1.25 | 模块名 `blog` |
| Web 框架 | Gin | 路由分组 + 自定义中间件链 |
| 命令行 | Cobra | `server` / `grpc` / `kafka-consume` / `migrate` |
| 关系型存储 | MySQL 8 + GORM | 用户、文章、评论、点赞、浏览历史 |
| 文档型存储 | MongoDB 7 | 站内通知 |
| 缓存 / 会话 | Redis 7 | Token 会话、点赞 Set、热榜 ZSet、分布式锁、nonce 去重 |
| 消息队列 | Kafka（segmentio/kafka-go） | 通知、浏览历史异步链路 + 死信队列 |
| 对象存储 | MinIO（minio-go） | 头像与正文图片预签名直传 |
| RPC | gRPC + Protocol Buffers | `proto/blogopen/v1`，生成代码在 `gen/` |
| 认证 | 自研 Redis Token + JWT（golang-jwt）+ HMAC-SHA256 | 分别用于站内、二方、三方 |
| 密码 | PBKDF2-SHA256 | 加盐迭代哈希 |
| 定时任务 | robfig/cron | 热榜每小时校准 |
| 参数校验 | validator/v10 | 含自定义校验规则 |
| 配置 | goccy/go-yaml | `config/config.yaml`、容器用 `config.docker.yaml` |
| IP 归属地 | ip2region | 离线 xdb，支持 IPv4 / IPv6 |
| 部署 | Docker、docker-compose、Kubernetes | 多阶段构建 + Alpine 运行镜像 |

## 架构与异步链路

```text
Client ──HTTP──> Gin(Handler) ──> Service ──> Repository ──> MySQL / MongoDB
                    │                │
                    │                ├──> Redis（会话 / 点赞 / 热榜 / 锁）
                    │                └──> MinIO（预签名直传）
                    │
                    └──> Kafka Producer ──> topic: notification / view_history
                                                   │
                                       kafka-consume 进程（批量消费 + 重试 + 死信）
                                                   ├──> 通知写入 MongoDB
                                                   └──> 浏览量 / 浏览历史写入 MySQL

Partner ──gRPC──> AuthInterceptor（JWT 或 HMAC）──> gRPC Handler ──> Service
```

启动流程：加载配置 → 初始化校验器与 IP 库 → 连接 MySQL / MongoDB / Redis / Kafka / MinIO → 组装 Repository、Service、Handler → 重建热榜缓存 → 启动定时任务 → 注册路由并监听端口。

## 接口一览

统一响应结构：

```json
{ "success": true, "code": 200, "message": "...", "data": {} }
```

需要登录的请求携带 `Authorization: Bearer <token>`；`/admin` 前缀额外要求管理员角色。

| 分组 | 方法 | 路径 | 说明 |
| --- | --- | --- | --- |
| 公开 | POST | `/user/register` | 注册 |
| 公开 | POST | `/user/login` | 登录并返回 Token |
| 公开 | GET | `/user/profile?user_id=` | 他人公开主页 |
| 公开 | GET | `/article/list` | 已发表文章列表 |
| 公开 | GET | `/article/hot-rank` | 热度榜 Top 10 |
| 公开 | GET | `/comment/list/roots` | 主评论列表 |
| 公开 | GET | `/comment/list/replies` | 子评论列表 |
| 可选登录 | GET | `/optional/article/detail?id=` | 文章详情，登录时返回 `is_liked` 并记录浏览历史 |
| 登录 | GET | `/auth/my/profile` | 我的主页 |
| 登录 | POST | `/auth/my/profile/update` | 修改昵称 / 头像 |
| 登录 | POST | `/auth/my/password/verify` | 校验旧密码换取一次性凭证 |
| 登录 | POST | `/auth/my/password/change` | 使用一次性凭证修改密码 |
| 登录 | POST | `/auth/my/account/update` | 修改手机号 |
| 登录 | POST | `/auth/my/avatar/upload-url` | 获取头像上传凭证 |
| 登录 | POST | `/auth/my/avatar/confirm` | 确认头像上传完成 |
| 登录 | POST | `/auth/my/logout` | 退出登录 |
| 登录 | POST | `/auth/comment/create` | 发表评论 / 回复 |
| 登录 | POST | `/auth/comment/delete` | 删除自己的评论 |
| 登录 | POST | `/auth/article/like`、`/auth/article/cancel_like` | 文章点赞 / 取消 |
| 登录 | POST | `/auth/comment/like`、`/auth/comment/cancel_like` | 评论点赞 / 取消 |
| 登录 | GET | `/auth/ntf/unread-count` | 未读通知数 |
| 登录 | GET | `/auth/ntf/list` | 通知列表 |
| 登录 | POST | `/auth/ntf/clear-unread` | 一键已读 |
| 管理员 | POST | `/admin/article/create` | 创建文章 |
| 管理员 | POST | `/admin/article/update` | 编辑文章 |
| 管理员 | POST | `/admin/article/publish` | 发布文章 |
| 管理员 | POST | `/admin/article/delete` | 软删除（进垃圾箱） |
| 管理员 | GET | `/admin/article/list` | 后台文章列表（按状态） |
| 管理员 | GET | `/admin/article/me/detail` | 后台文章详情 |
| 管理员 | GET | `/admin/article/trash/list` | 垃圾箱列表 |
| 管理员 | POST | `/admin/article/trash/recover` | 恢复文章 |
| 管理员 | POST | `/admin/article/trash/clear` | 彻底删除 |
| 管理员 | POST | `/admin/article/image/upload-url` | 正文图片上传凭证 |
| 管理员 | POST | `/admin/comment/delete` | 删除任意评论 |

gRPC 服务（默认端口 `9100`，定义见 `proto/blogopen/v1`）：

| 服务 | 方法 | 开放对象 |
| --- | --- | --- |
| `UserService` | `GetUserBasicInfo` | 二方 |
| `UserService` | `GetPublicUserInfo` | 三方 |
| `ArticleService` | `GetAvailableList` | 二方 |
| `CommentService` | `GetCommentStats` | 二方 |

## 项目结构

```text
.
├── cmd/                    # Cobra 命令：server / grpc / kafka-consume / migrate
├── config/                 # YAML 配置结构与加载（含 Kafka 配置）
├── deploy/k8s/             # Kubernetes 清单（命名空间、配置、中间件、三个工作负载）
├── gen/                    # protoc 生成的 gRPC 代码
├── proto/blogopen/v1/      # 开放 API 的 proto 定义
├── internal/
│   ├── auth/               # Token 会话、JWT、HMAC、密码哈希
│   ├── common/             # 统一响应、错误码、校验器、防重、浏览窗口
│   ├── consts/             # Redis key 与锁常量
│   ├── cron/               # 定时任务管理器与热榜校准
│   ├── dto/                # 各模块请求 / 响应模型
│   ├── grpc/               # gRPC server、handler、认证拦截器
│   ├── handler/            # HTTP 处理器
│   ├── message/            # Kafka 消息体定义
│   ├── middleware/         # 认证、管理员校验、日志、异常、防重、浏览历史
│   ├── model/              # 持久化模型与状态 / 角色枚举
│   ├── mq/                 # Kafka 消息处理器与注册表
│   ├── repository/         # MySQL / MongoDB 数据访问
│   ├── routes/             # 公开 / 登录 / 管理员 / 可选登录路由
│   └── service/            # 业务规则、事务、缓存、异步投递
├── pkg/
│   ├── database/           # MySQL、MongoDB、Redis 客户端
│   ├── kafka/              # 客户端、生产者、消费者、死信队列
│   ├── oss/                # MinIO 客户端
│   ├── resource/           # ip2region 离线库
│   └── util/               # 缓存、IP、Redis 锁工具
└── scripts/                # MySQL 建库建表脚本、MongoDB 初始化脚本
```

## 快速开始

### 本地运行

```bash
# 1. 准备配置：按需修改数据库、Redis、MongoDB、Kafka、MinIO、JWT 等配置
vim config/config.yaml

# 2. 初始化数据库（自动执行 scripts/mysql 下的建库建表脚本）
go run . migrate

# 3. 启动 HTTP 服务（默认 8080）
go run . server -p 8080

# 4. 启动 Kafka 消费者（通知与浏览历史落库）
go run . kafka-consume

# 5. 按需启动开放 API 服务（默认读取 config 中的 grpc.port）
go run . grpc
```

MongoDB 通知集合的索引初始化脚本见 `scripts/mongodb/notification.js`。

### 容器与集群

```bash
# 一键拉起 MySQL / Redis / MongoDB / Kafka 与三个服务进程
docker compose up -d

# 生产编排
docker compose -f docker-compose.prod.yml up -d

# Kubernetes：按文件名序号顺序应用
kubectl apply -f deploy/k8s/
```

### 测试

```bash
go test ./...
go test ./... -cover
```
