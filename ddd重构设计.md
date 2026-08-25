# 基于单体分支的 DDD 重构设计文档

## 1. 文档目的

本文档用于说明：以 `70b6558709e587ab1408375d1598b6b9e192e770` 为基础的大单体分支，如何在**不增加、不减少现有系统功能**的前提下，将项目重构为标准的 DDD 四层架构。

本次重构的目标不是重新设计业务，也不是拆分微服务，而是：

1. 保留当前系统已有的全部功能、接口、数据结构和异步链路。
2. 将业务代码整理为五个独立的 DDD 上下文：用户、文章、评论、点赞、通知。
3. 在每个上下文内部采用 `Interfaces → Application → Domain` 的依赖方向。
4. 由 `Infrastructure` 实现 `Domain` 定义的 Port。
5. 保持原有代码中的业务处理顺序和中文注释风格。
6. 保持大单体运行方式，不保留 Identity、Content、Community 三个微服务进程划分。

***

## 2. 重构范围

### 2.1 本次重构包含的内容

- 从传统的 `service / repository / model / handler` 结构迁移到上下文内聚的 DDD 结构。
- 将用户、文章、评论、点赞、通知分别整理为五个上下文。
- 将应用服务、领域模型、Repository Adapter、HTTP/gRPC/Kafka Adapter 放入对应上下文。
- 重新整理大单体的组合根和依赖注入过程。
- 将上下文之间的直接 Repository 依赖改为 Port、Application Facade 或防腐层适配器。
- 保留 HTTP、对外 gRPC、Kafka、Cron、数据库、缓存和对象存储等现有能力。
- 更新架构说明、目录说明和启动说明。

### 2.2 本次重构不包含的内容

以下内容不得在本次重构中新增、删除或改变：

- 不新增业务功能。
- 不删除已有业务功能。
- 不改变现有 HTTP 路由、请求字段、响应字段和错误码。
- 不改变现有对外 gRPC Service、Method、Request 和 Response 契约。
- 不改变 Kafka Topic、Consumer Group 和既有消息字段。
- 不改变 MySQL、MongoDB、Redis、MinIO 中已有数据的业务含义。
- 不调整业务规则、权限规则、状态机和计算公式。
- 不以重构为理由修改原有业务代码的执行顺序。
- 不以重构为理由统一或重写既有代码注释风格。
- 不引入新的微服务进程、服务发现、内部网络调用或独立部署单元。

***

## 3. 目标运行形态

重构后的项目仍然是一个大单体项目。

```text
                          ┌──────────────────────┐
                          │      blog 单体程序     │
                          └──────────┬───────────┘
                                     │
          ┌──────────────────────────┼──────────────────────────┐
          ▼                          ▼                          ▼
    server 子命令                grpc 子命令              kafka-consume 子命令
    HTTP 接口入口                对外 gRPC 入口              Kafka 消费入口
          │                          │                          │
          └──────────────────────────┼──────────────────────────┘
                                     ▼
                          五个上下文的 Application
                                     │
        ┌────────────────┬───────────┼───────────┬────────────────┐
        ▼                ▼           ▼           ▼                ▼
      User           Article      Comment       Like         Notification
```

`server`、`grpc` 和 `kafka-consume` 可以继续作为同一个二进制的不同子命令存在。它们是不同的运行入口，即可以使用cobra单独开启在不同端口

### 3.1 允许保留的技术能力

以下能力继续保留：

- HTTP 服务。
- 对外开放 gRPC 服务。
- Kafka 生产和消费。
- 通知异步处理。
- 浏览历史异步处理。
- MySQL 持久化。
- MongoDB 通知存储。
- Redis 缓存、分布式锁和会话。
- MinIO 对象存储。
- Cron 定时任务。
- 数据库迁移命令。

#

***

## 4. 五个 DDD 上下文

本项目最终采用五个上下文：

| 上下文   | 目录                      | 主要职责                     |
| ----- | ----------------------- | ------------------------ |
| 用户上下文 | `internal/user`         | 注册、登录、会话、用户资料、头像、密码修改    |
| 文章上下文 | `internal/article`      | 文章生命周期、文章查询、图片、浏览历史、热榜   |
| 评论上下文 | `internal/comment`      | 评论、回复、评论删除、评论查询和评论统计     |
| 点赞上下文 | `internal/like`         | 文章点赞、评论点赞、取消点赞、点赞状态和点赞关系 |
| 通知上下文 | `internal/notification` | 通知创建、通知查询、未读状态、通知消费      |

五个上下文是业务领域边界，不等于五个独立进程，也不等于五个数据库实例。

### 4.1 用户上下文

用户上下文负责：

- 用户注册。
- 手机号或昵称登录。
- 密码校验和密码修改。
- Token 会话创建、查询和失效。
- 用户公开资料和个人资料。
- 头像上传凭证和头像确认。
- 用户角色和权限相关的领域数据。

用户上下文不得直接依赖文章、评论、点赞和通知 Repository。

### 4.2 文章上下文

文章上下文负责：

- 文章创建、编辑、发布、软删除、恢复和彻底删除。
- 文章状态机。
- 文章列表、详情和分页查询。
- 文章正文图片处理。
- 浏览量和浏览历史。
- 热度计算和热榜维护。
- 文章相关的读模型和统计投影。

文章上下文需要用户信息时，只依赖用户查询 Port，不直接访问用户 Repository。

### 4.3 评论上下文

评论上下文负责：

- 主评论和回复。
- 评论分页和查询。
- 评论作者信息组装所需的查询。
- 评论删除和级联规则。
- 回复目标校验。
- 评论数量统计。
- 评论创建事件发布。

评论上下文需要校验文章时，只依赖文章存在性或可评论性 Port。

### 4.4 点赞上下文

点赞上下文负责：

- 文章点赞和评论点赞。
- 点赞取消。
- 重复点赞和重复取消的幂等行为。
- 点赞目标类型和目标 ID 校验。
- 点赞关系持久化。
- 点赞状态查询。
- 点赞事件发布。

点赞上下文不得直接注入文章 Repository 或评论 Repository。目标验证通过领域 Port 完成。

### 4.5 通知上下文

通知上下文负责：

- 通知创建。
- 通知列表查询。
- 未读数量查询。
- 一键已读或清理未读状态。
- 点赞、评论、回复等通知事件消费。
- Kafka 消息幂等和失败处理。

通知上下文需要文章、评论或用户的展示信息时，只依赖最小查询 Port，不拥有和修改对方上下文的领域数据。

***

## 5. 目标目录结构

### 5.1 业务上下文目录

```text
internal
├── article
│   ├── app
│   │   ├── dto
│   │   ├── engagement_service.go
│   │   ├── rank.go
│   │   ├── service.go
│   │   └── views.go
│   ├── domain
│   │   ├── article.go
│   │   ├── errors.go
│   │   ├── hot_rank.go
│   │   ├── ports.go
│   │   └── view_history.go
│   ├── infra
│   │   ├── model
│   │   ├── article_repository.go
│   │   ├── article_image_storage.go
│   │   ├── hot_rank_store.go
│   │   ├── ranking_query.go
│   │   ├── statistics.go
│   │   ├── view_event_publisher.go
│   │   └── view_history_repository.go
│   └── interfaces
│       ├── http
│       ├── grpc
│       └── kafka
│
├── comment
│   ├── app
│   │   ├── dto
│   │   ├── comments.go
│   │   ├── like_projection.go
│   │   └── service.go
│   ├── domain
│   │   ├── comment.go
│   │   ├── errors.go
│   │   └── ports.go
│   ├── infra
│   │   ├── model
│   │   ├── article_query.go
│   │   ├── comment_repository.go
│   │   ├── event_publisher.go
│   │   └── statistics.go
│   └── interfaces
│       ├── http
│       ├── grpc
│       └── kafka
│
├── like
│   ├── app
│   │   ├── dto
│   │   ├── like_service.go
│   │   └── service.go
│   ├── domain
│   │   ├── like.go
│   │   ├── ports.go
│   │   └── repositories.go
│   ├── infra
│   │   ├── model
│   │   ├── event_publisher.go
│   │   ├── like_cache.go
│   │   └── like_repositories.go
│   └── interfaces
│       ├── http
│       ├── grpc
│       └── kafka
│
├── notification
│   ├── app
│   │   ├── dto
│   │   └── service.go
│   ├── domain
│   │   ├── notification.go
│   │   └── ports.go
│   ├── infra
│   │   ├── model
│   │   ├── notification_repository.go
│   │   └── queries.go
│   └── interfaces
│       ├── http
│       ├── grpc
│       └── kafka
│
└── user
    ├── app
    │   ├── dto
    │   └── service.go
    ├── domain
    │   ├── errors.go
    │   ├── password.go
    │   ├── ports.go
    │   ├── session.go
    │   └── user.go
    ├── infra
    │   ├── model
    │   ├── avatar_storage.go
    │   ├── password_change_token_store.go
    │   ├── token_session.go
    │   └── user_repository.go
    └── interfaces
        ├── http
        ├── grpc
        └── kafka
```

实际文件迁移时，以当前代码中的文件职责为准，不要求为了目录整齐而改变现有函数顺序、业务逻辑顺序或注释顺序。

### 5.2 平台和共享目录

业务上下文之外只保留真正的公共能力：

```text
internal
├── platform
│   ├── auth
│   ├── bootstrap
│   ├── config
│   ├── cron
│   └── interfaces
└── shared
    ├── common
    └── consts
```

其中：

- `platform/auth`：认证、JWT、HMAC、密码工具等通用技术能力。
- `platform/bootstrap`：配置、数据库、Redis、Kafka、MinIO 等客户端初始化。
- `platform/config`：业务配置读取。
- `platform/cron`：定时任务调度和任务注册。
- `platform/interfaces`：仅保留无法归属于某个上下文的统一入口、路由装配和通用中间件。
- `shared/common`：统一响应、错误、校验、防重复提交等真正跨上下文的通用能力。
- `shared/consts`：跨上下文且稳定的公共常量。

不得将业务 Service、业务 Repository 或业务领域规则放入 `platform` 或 `shared`。

***

## 6. 四层架构约束

每个上下文内部均采用以下依赖关系：

```text
Interfaces → Application → Domain
Infrastructure → 实现 Domain 定义的 Port
```

### 6.1 Interfaces 层

职责：

- 接收 HTTP、gRPC、Kafka 等外部输入。
- 完成协议参数转换。
- 调用 Application Service。
- 将应用错误转换为 HTTP、gRPC 或消息处理结果。
- 处理入口级鉴权、参数校验和日志。

限制：

- 不直接访问 Repository。
- 不编写领域规则。
- 不直接操作 GORM、MongoDB、Redis 或 Kafka 客户端完成业务逻辑。

### 6.2 Application 层

职责：

- 编排业务用例。
- 调用领域对象和 Domain Port。
- 处理事务边界、幂等和跨上下文协作。
- 组装 DTO 和响应模型。
- 调用领域服务、Repository Port 和查询 Port。

限制：

- 不依赖 Gin、GORM、Redis、MongoDB、Kafka、MinIO 等技术框架。
- 不直接创建基础设施客户端。
- 不直接依赖其他上下文的 Repository。

### 6.3 Domain 层

职责：

- 聚合根、实体和值对象。
- 领域状态和领域行为。
- 领域错误。
- 领域事件。
- Repository Port、查询 Port、外部服务 Port。

限制：

- 不导入 GORM。
- 不导入 Redis 客户端。
- 不导入 MongoDB 客户端。
- 不导入 Kafka 客户端。
- 不导入 MinIO 客户端。
- 不导入 Gin。
- 不导入 gRPC 框架。
- 不依赖其他上下文的 Infrastructure 或 Interfaces。

### 6.4 Infrastructure 层

职责：

- 实现 Domain 定义的 Repository Port。
- 实现数据库、缓存、对象存储、消息发布等 Adapter。
- 实现调用方上下文定义的跨上下文查询 Port。
- 负责技术模型与领域模型之间的转换。

限制：

- 不向上层泄露 GORM Model、MongoDB Document 或 Redis 数据结构。
- 不把基础设施客户端直接传入 Domain。
- 不跨上下文直接写入其他上下文的数据表。

***

## 7. 上下文之间的协作方式

### 7.1 基本原则

五个上下文之间禁止直接依赖对方的：

```text
Repository
Infrastructure Model
Application 内部实现
HTTP Handler
```

跨上下文只能依赖对方提供的最小能力接口：

```text
Query Port
Application Facade
Domain Event
ACL Adapter
```

### 7.2 文章与用户

文章需要作者信息时：

```text
Article Application
        ↓
Article UserInfoQuery Port
        ↓
User Application Facade
```

文章不直接依赖：

```text
user/infra/user_repository.go
```

### 7.3 评论与文章

评论创建前需要判断文章是否存在或是否允许评论：

```text
Comment Application
        ↓
Comment ArticleQuery Port
        ↓
Article Application Facade
```

评论只获取文章的最小查询结果，不修改文章聚合。

### 7.4 点赞与文章、评论

点赞目标校验通过统一的目标查询 Port：

```text
Like Application
        ↓
Like TargetQuery Port
        ├── ArticleTargetQuery
        └── CommentTargetQuery
```

在大单体中，这些 Adapter 通过本地 Application Facade 实现，不使用内部 gRPC。

### 7.5 通知与其他上下文

通知可以通过两种方式获得数据：

1. 消费现有 Kafka 事件。
2. 通过最小查询 Port 获取展示所需的用户、文章或评论信息。

通知上下文不直接写入用户、文章、评论和点赞的数据。

***

## 9. 代码迁移原则

### 9.1 以 70b 分支作为行为基线

以 `70b6558709e587ab1408375d1598b6b9e192e770` 作为大单体基础分支，保留：

- 原有配置结构。
- 原有启动命令。
- 原有数据库连接方式。
- 原有中间件客户端。
- 原有路由和接口契约。
- 原有业务功能。

当前五个上下文目录作为 DDD 代码迁移来源，不直接覆盖当前分支。

### 9.2 按上下文逐步迁移

推荐迁移顺序：

```text
User
  ↓
Article
  ↓
Comment
  ↓
Like
  ↓
Notification
  ↓
统一接口和组合根
```

每迁移一个上下文，必须同时迁移：

```text
domain
app
infra
interfaces
相关测试
```

不得只迁移业务 Service 而暂时保留旧 Repository 作为长期依赖。

### 9.3 保留代码顺序

重构时应遵守：

1. 原函数内部的业务处理步骤不重新排序。
2. 原有校验顺序不改变。
3. 原有数据库操作顺序不改变。
4. 原有事件发布时机不改变。
5. 原有缓存更新时机不改变。
6. 原有错误返回时机不改变。
7. 原有注释内容和注释位置尽量原样保留。
8. 只在包路径、依赖注入和职责归属必要时进行最小调整。

### 9.4 兼容适配器只作为迁移工具

如果迁移过程中需要暂时兼容旧的 `internal/service` 或 `internal/repository`，可以增加短期 Adapter，但必须：

- 明确标记迁移用途。
- 不让新的 Domain 依赖旧 Service。
- 不让新 Application 长期依赖旧 Repository。
- 在对应上下文迁移完成后删除兼容代码。

***

## 10. 功能不变的验收标准

### 10.1 HTTP 契约

- 所有路由路径保持不变。
- 所有 HTTP 方法保持不变。
- 请求参数名称和类型保持不变。
- 响应 JSON 字段保持不变。
- 统一响应结构保持不变。
- 错误码和错误信息映射保持不变。
- 游客、用户和管理员权限行为保持不变。

### 10.2 对外 gRPC 契约

- Service 名称不变。
- RPC Method 名称不变。
- Request 字段不变。
- Response 字段不变。
- JWT 和 HMAC 鉴权行为不变。
- 错误码映射保持不变。

### 10.3 Kafka 契约

- Topic 名称不变。
- Consumer Group 保持兼容。
- 现有 JSON 消息字段保持兼容。
- 通知消费行为不变。
- 浏览历史消费行为不变。
- 重试和死信行为不变。
- 重复消息处理行为不变。

### 10.4 数据行为

- 不修改已有数据库表结构，除非是当前功能已经要求的兼容性变更。
- 不改变 MySQL 表字段含义。
- 不改变 MongoDB 通知文档含义。
- 不改变 Redis Key 规则。
- 不改变 MinIO 文件路径和上传行为。
- 不新增数据迁移要求。

### 10.5 架构检查

必须满足：

```text
Domain 不依赖技术框架
Application 不依赖技术框架
Interfaces 不直接访问 Repository
跨上下文不直接依赖 Repository
Infrastructure 只实现 Port 和技术 Adapter
```

建议持续执行：

```bash
go test ./...
sh scripts/check_architecture.sh
```

***

## 11. 目标结果

重构完成后的代码结构应当表达以下关系：

```text
五个业务上下文
        ↓
每个上下文内部完整四层
        ↓
同一进程内统一装配
        ↓
跨上下文通过 Port / Facade / Event 协作
        ↓
HTTP、对外 gRPC、Kafka 保持原功能
```

最终目标不是简单把文件移动到五个目录，而是让目录、依赖和运行方式同时表达同一套架构：

```text
按业务上下文组织代码
按四层架构限制依赖
按大单体方式统一运行
按原有行为进行兼容验收
```

