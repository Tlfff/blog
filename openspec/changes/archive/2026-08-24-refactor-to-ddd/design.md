## Context

当前单体以 `handler / service / repository / model` 技术分层组织，三个 Cobra 子命令分别装配 HTTP、对外 gRPC 和 Kafka Consumer。业务 Service 直接持有多个上下文的 Repository、Redis、Kafka、MinIO 或 MongoDB 客户端；部分 Repository 通过 JOIN 读取用户展示信息；点赞和评论用例会在同一 MySQL 事务中同时修改关系/评论表与文章统计字段。

本次以 `70b6558709e587ab1408375d1598b6b9e192e770` 为唯一行为基线。当前分支中的设计文件以及 `feature/DDD`、`82aa39e` 只能用来参考目录和局部领域代码，不得改变基线功能、执行顺序或运行形态。项目继续作为一个二进制、多个子命令运行，共享现有 MySQL、MongoDB、Redis、Kafka 和 MinIO。

本 Change 通过 `modular-ddd-architecture` delta spec 将关键架构边界和兼容性约束转化为可验证要求；该 spec 不代表新增业务功能。

## Goals / Non-Goals

**Goals:**

- 建立 User、Article、Comment、Like、Notification 五个业务上下文及清晰的数据写入所有权。
- 形成 `Interfaces → Application → Domain` 的业务依赖方向，由 Infrastructure 实现 Domain/Application 定义的 Port。
- 保留现有跨表写入的本地 MySQL 事务原子性，同时消除 Application 对 GORM 和其他上下文 Repository 的直接依赖。
- 保留复杂列表查询的 SQL 性能，并阻止专用只读 JOIN 演变为跨上下文写入或共享 Repository。
- 逐上下文迁移并持续接入组合根，避免完成所有文件移动后才首次集成。
- 将仅供本项目使用的 `pkg` 技术能力收敛到 `internal/platform`。
- 建立适合新架构的领域、应用、Adapter、契约和架构依赖测试。

**Non-Goals:**

- 不新增、删除或补齐任何业务功能、事件、通知类型、目标校验、消费幂等或投影能力。
- 不拆分微服务，不新增内部 gRPC、服务发现、独立数据库或部署单元。
- 不修改 HTTP、对外 gRPC、Kafka、数据库、Redis Key 或 MinIO 对象路径契约。
- 不要求每个上下文拥有完全对称的目录，也不为一一对应的简单数据结构创建无语义 Mapper。
- 不要求保留内部包名、类型名、方法名、日志原文或与旧目录强耦合的测试文件。

## Decisions

### 1. 采用五上下文模块化单体，并明确数据写入所有权

五个上下文的数据所有权如下：

| 上下文 | 拥有的写模型和职责 |
| --- | --- |
| User | `users`、会话、密码修改凭证、头像流程、角色与状态 |
| Article | `articles`、文章图片、浏览历史、浏览量、文章统计字段、热榜 |
| Comment | `comments`、回复关系、评论状态与根评论回复计数 |
| Like | `article_likes`、`comment_likes`、点赞关系缓存 |
| Notification | MongoDB `notifications`、当前已接线的通知消费与查询 |

一个上下文不得直接使用 SQL、Repository 或持久化模型写入另一个上下文的数据。需要更新对方拥有的数据时，由调用方 Application 依赖最小 Port，再由本地 Adapter 调用对方 Application Facade。

选择五个上下文而不是把 Like 合并进 Article/Comment，是因为当前存在两套独立关系表、幂等状态和缓存行为；独立边界可以集中互动关系逻辑。选择模块化单体而不是微服务，是为了保留当前本地调用、共享数据库事务和单二进制运行成本。

### 2. 目录按上下文组织，Adapter 按实际能力创建

每个上下文使用以下基本形态，但只创建实际需要的文件和 Adapter：

```text
internal/<context>
├── domain
│   ├── <aggregate>.go
│   ├── status.go
│   ├── errors.go
│   └── repository.go
├── application
│   ├── command.go
│   ├── query.go
│   ├── service.go
│   └── ports.go
├── infrastructure
│   ├── <repository-or-adapter>.go
│   └── model/                 # 仅在确有独立持久化模型时建立
└── interfaces
    ├── http/                  # 有 HTTP 入口时建立
    ├── grpc/                  # 有 gRPC 入口时建立
    └── kafka/                 # 有 Kafka 消费入口时建立
```

例如 Like 当前只需要 HTTP Interfaces，Notification 需要 HTTP 和 Kafka，不能为了目录对称创建空 gRPC 包。上下文根部可以提供显式模块构造函数，负责组合本上下文各层并暴露必要的 Application Facade 和入口 Adapter；业务代码不得通过全局容器或字符串进行服务查找。

### 3. 区分 Domain Port 与 Application Port

Domain 只定义领域模型执行规则所必需的抽象，主要包括聚合持久化 Repository 及真正的领域协作者。Application 在 `application/ports.go` 中定义用例编排所需的出站能力，包括：

- 跨上下文查询与写入 Facade；
- TransactionManager；
- Cache；
- Kafka Publisher；
- MinIO Storage；
- Session Store 和密码修改凭证 Store；
- Clock 等用例级外部能力。

Port 由调用方按其最小需求定义，Adapter 也归调用方 Infrastructure/ACL 所有。例如 Like 定义 `ArticleTargetQuery` 和 `ArticleEngagementWriter`，Like 的 Adapter 调用 Article Application Facade，而不是由 Like 导入 Article Repository。

相比把所有 Port 放进 Domain，此方案避免 Domain 知道 Kafka、对象存储、展示查询和跨上下文编排；相比集中建立一个共享大接口，此方案能够保持接口最小并防止上下文隐式耦合。

### 4. TransactionManager 属于 Application Port，GORM 事务属于 Platform Infrastructure

事务边界由 Application 用例决定，因为它描述的是“一次用例中的哪些持久化操作必须共同成功或回滚”，而不是单个领域对象的业务规则。Domain 只决定状态转换是否合法，不负责 `BEGIN / COMMIT / ROLLBACK`、跨上下文调用顺序或持久化资源传播。

每个需要事务的 Application 以最小接口声明 `TransactionManager`。`internal/platform/transaction` 提供共享 GORM 实现：

1. `WithinTransaction` 使用根 `*gorm.DB` 开启事务；
2. 将当前事务资源写入派生 `context.Context`，context key 对业务层不可见；
3. Like、Comment、Article 等 Repository Adapter 通过 Platform 的事务解析器优先取得当前事务，否则使用默认 DB；
4. 调用其他 Application Facade 时原样传递事务 context，使不同上下文的 Adapter 参加同一个本地 MySQL 事务；
5. Application 和 Domain 均不导入 GORM。

文章点赞继续保持以下边界：

```text
TransactionManager.WithinTransaction
├── LikeRepository 保存点赞关系
└── Article Application Facade 更新 articles.like_count
提交成功
├── 更新 Redis 点赞缓存
└── 按现有时机发布通知消息
```

评论创建、主评论删除和子评论删除同样在事务内通过 Article Application Facade 更新 `articles.comment_count`。事务外的缓存和 Kafka 操作仍按基线顺序执行，不扩展事务范围，也不引入 Outbox 或最终一致事件。

选择 context 传播是为了避免将 `*gorm.DB` 放入 Application Port 方法签名。其代价是事务参与关系不如显式参数直观，因此只允许 Platform 事务包和 Infrastructure Adapter 读取事务资源，并通过测试验证同一事务及回滚行为。

### 5. 跨上下文读取采用 Read Model JOIN 与 Query Facade 的混合方案

#### 复杂列表和展示查询

文章列表、评论列表等需要分页、排序并组装用户昵称/头像的查询，由调用方定义 Read Model Query Port，并在调用方 Infrastructure 中实现专用只读 JOIN：

```text
Article Application → ArticleListQuery → article/infrastructure/article_list_query.go
Comment Application → CommentListQuery → comment/infrastructure/comment_list_query.go
```

这些 Adapter 可以读取其他上下文的表，但必须满足：

- 只能执行只读查询；
- 只能返回调用方定义的 Read Model；
- 不返回或复用对方聚合、Repository 或 Infrastructure Model；
- SQL 中涉及的跨上下文字段必须限于当前响应真正需要的字段；
- 不作为任何写入路径或领域规则入口。

该选择保留当前 JOIN、分页、排序和查询次数，优先保证模块化单体中的性能与行为兼容。

#### 规则校验和最小对象查询

文章存在性、评论可互动性、单个用户公开资料、通知生成所需的最小文章/评论信息等，使用调用方 Application Query Port，由本地 Adapter 调用对方 Application Facade。批量组装但不依赖跨域排序时允许使用 `BatchGetPublicProfiles`，严禁逐条查询形成 N+1。

相比所有查询都严格经过 Facade，混合方案保留复杂列表性能；相比所有读取都直接 JOIN，它又能让规则校验和业务语义继续由数据所有者提供。

### 6. Domain Model 与持久化 Model 采用务实复用策略

当 MySQL 表字段与领域对象基本一一对应时，允许 Domain Model 携带 GORM struct tag，以避免只有字段复制、没有语义价值的双模型和 Mapper。Domain Model 仍不得：

- 导入 GORM；
- 包含 `*gorm.DB`、SQL 表达式或 Repository 实现；
- 暴露只为跨上下文查询服务的 JOIN 字段；
- 承担 HTTP、gRPC 或 Kafka 协议 DTO 职责。

以下情况必须创建 Infrastructure Model/Read Model 并进行显式转换：

- Notification 文档包含 MongoDB `primitive.ObjectID` 等技术类型；
- 存储结构与领域结构明显不同；
- 复杂 JOIN、聚合统计或分页投影；
- 技术字段不应进入领域语义；
- 一个领域对象需要多种不同持久化表示。

协议 DTO 留在 Interfaces；Application 使用与 Gin、protobuf、Kafka SDK 无关的 Command、Query 和 Result 类型。

### 7. 组合根先建立骨架，再逐上下文接入

组合根分为两部分：

1. `internal/platform/bootstrap` 只初始化和关闭配置、MySQL、MongoDB、Redis、Kafka、MinIO 等技术资源；
2. 各上下文 Module 构造函数接收所需资源和跨上下文 Port，构造 Domain/Application/Infrastructure/Interfaces，并显式暴露入口和 Facade。

迁移初期建立统一资源骨架，但不一次性切换全部业务。迁移每个上下文时：

- 构造新 Module；
- 接入对应 HTTP、gRPC、Kafka 或 Cron 入口；
- 未迁移上下文继续使用旧实现；
- 必要时使用标记清晰的短期兼容 Adapter；
- 新上下文验收后删除该上下文的旧 Service、Repository、Model、Handler 及兼容代码。

`cmd/server.go`、`cmd/grpc.go`、`cmd/kafka_consume.go` 只选择所需资源和 Module，不自行构造具体 Repository。最后统一路由与消费者注册，删除旧装配路径。不同子命令不强制初始化其不需要的客户端。

相比最后一次性集成，该方案能尽早发现模块构造和循环依赖问题；相比长期保留两套实现，它要求每个上下文完成后立即收敛旧路径。

### 8. Platform 与安全能力归属

`pkg` 下仅供本项目使用的技术代码迁入：

```text
internal/platform
├── bootstrap
├── config
├── database
├── transaction
├── kafka
├── redis
├── oss
├── security
├── cron
└── interfaces
    ├── http
    └── grpc
```

User Domain/Application 拥有密码规则与校验语义、Session 生命周期、密码修改凭证、角色和状态，并声明 `PasswordHasher`、`SessionStore`、`PasswordChangeTokenStore`、`AvatarStorage` 等 Port。

`platform/security` 提供 PBKDF2、JWT 和 HMAC 技术实现；Redis Session Store 是 User Application Port 的 Adapter；HTTP 鉴权中间件、gRPC JWT/HMAC Interceptor 和统一协议错误映射属于 `platform/interfaces`。技术实现不得反向拥有用户业务规则。

`internal/shared` 只保留无上下文所有权、稳定且纯粹的值类型或极小工具。现有 `common` 和 `consts` 不整体搬迁：领域错误与 Redis Key 分别下沉到其所属上下文，HTTP/gRPC 响应及错误映射放入 Platform Interfaces，Kafka/Redis 技术错误放入相应 Platform 包。

### 9. 只迁移基线中已存在且已接线的能力

实现时必须从 `70b6558` 逐项建立能力清单和调用链。以下内容即使在旧模型中存在常量、未使用方法或参考分支代码，也不得据此补齐：

- 评论创建事件；
- 评论、回复、评论点赞通知链路；
- Kafka event ID、version、envelope；
- 新的业务消费幂等；
- 新的点赞目标存在性校验；
- 异步统计投影或 Outbox；
- 参考分支中的内部 gRPC 和微服务部署。

未接线代码是否保留，应以它是否构成 `70b6558` 可达运行行为为准；不能把参考分支实现当作新增功能来源。

### 10. 测试采用“逐上下文建立替代安全网，再删除旧测试”的策略

不采用“一开始删除全部旧测试，迁移结束后再补”的方案。该方案迁移成本表面较低，但会在最长的重构阶段失去回归定位能力，并且与“行为以 `70b6558` 为基线”相冲突。

也不要求先修复和保留所有旧测试。采用折中策略：

1. 迁移某上下文前，记录该上下文的路由、DTO、错误码、SQL副作用顺序、Kafka消息和对外 gRPC 行为；
2. 保留仍有效且成本合理的现有测试，补充覆盖关键基线行为的最小 Characterization/Golden Test；
3. 实现 Domain 单元测试、Application Port 测试、事务回滚测试和 Adapter 集成测试；
4. 新实现达到等价覆盖后，删除依赖旧构造函数、旧包路径或错误测试夹具的测试；
5. 每个上下文迁移完成时运行该上下文测试、架构检查和全量可运行测试。

当前已知的 Validator 未注册和 IP 资源相对路径问题，应作为测试夹具问题处理：若对应行为仍需覆盖，则修复或由等价新测试替代；不得为了得到全绿结果而删除未被新测试覆盖的关键兼容行为。

### 11. 使用自动化架构检查约束依赖方向

新增或完善架构检查，至少验证：

- Domain/Application 不导入 Gin、GORM、Redis、MongoDB、Kafka、MinIO、gRPC 等技术 SDK；
- Interfaces 不直接导入业务 Repository 或操作技术客户端完成业务逻辑；
- 一个上下文不导入其他上下文的 Infrastructure、Interfaces 或 Repository；
- 跨上下文写入只能经过允许的 Application Port/Facade；
- `pkg` 迁移完成后业务代码不再依赖旧技术包路径；
- 旧 `internal/service`、`internal/repository`、`internal/model`、`internal/handler` 等目录最终清空或删除。

架构脚本是迁移门禁，不替代行为测试。

## Risks / Trade-offs

- **[事务通过 context 隐式传播，参与关系不够直观]** → 仅允许 Platform Transaction 和 Infrastructure Adapter 操作事务 context；为点赞和评论编写提交、回滚及跨上下文同事务测试。
- **[专用 Read Model JOIN 仍耦合其他上下文的数据库 Schema]** → 仅允许只读、调用方拥有、字段最小化的 Query Adapter；禁止复用对方模型和写表，并用架构检查限制用途。
- **[逐步迁移期间新旧架构并存，可能产生双重装配]** → 每个入口只能注册一套实现；上下文切换后立即删除其旧装配和短期 Adapter，禁止长期双写或双路由。
- **[务实复用 Domain/MySQL Model 会保留 GORM tag]** → 允许声明式 tag，但禁止技术类型和技术 API 进入 Domain；出现存储差异时立即拆分 Infrastructure Model。
- **[移动 `pkg` 影响面大，可能掩盖业务迁移问题]** → Platform 骨架先建立兼容包路径或小步移动，按技术能力逐项替换 import，并在五上下文完成前收敛旧路径。
- **[删除旧测试可能降低行为覆盖]** → 只在对应 Characterization、Application 或 Adapter 测试已提供等价覆盖后删除；每个上下文单独审查测试映射。
- **[参考分支包含超范围功能，复制代码可能引入行为变化]** → 所有参考代码必须对照 `70b6558` 的可达调用链、协议和副作用顺序审查，不能按目录批量合并。
- **[内部重命名可能导致错误映射或中间件顺序变化]** → 通过 HTTP/gRPC/Kafka 契约测试及入口装配测试固定外部行为与中间件顺序。

## Migration Plan

1. 固定 `70b6558` 的行为和依赖清单，记录 HTTP、gRPC、Kafka、Cron、表/集合、Redis Key、MinIO 路径及关键事务顺序。
2. 建立 Platform Resources、TransactionManager、模块构造约定和架构检查骨架，但保持旧业务入口继续运行。
3. 按 `User → Article → Comment → Like → Notification` 顺序迁移；每个上下文均完成 Domain、Application、Infrastructure、真实存在的 Interfaces、测试和组合根接入后再进入下一上下文。
4. Article 迁移时建立 User 查询 Adapter 和 Article Read Model JOIN；Comment 迁移时建立 Article Facade、用户展示 Read Model；Like 迁移时接入 Article/Comment Facade 和跨上下文事务；Notification 最后接入当前实际存在的 Kafka 消费链路。
5. 将 Cron、统一 HTTP/gRPC 中间件和路由/消费者注册切换到新模块；迁移 `pkg` 技术能力并移除旧 import。
6. 删除旧 Service、Repository、Model、Handler、Routes/MQ 装配和所有短期兼容 Adapter，执行全量测试、架构检查及契约对比。
7. 部署方式、配置和数据不变。若某阶段验收失败，回退该上下文的组合根接线和对应提交；不存在数据库迁移，因此不需要数据回滚。最终切换失败时回退到最后一个已通过上下文验收的提交。
