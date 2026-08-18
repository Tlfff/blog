## Context

当前项目是一个 Go 单体应用，使用 Gin、Cobra、GORM、Redis、MongoDB、Kafka 和 MinIO。`server`、`grpc` 和 `kafka-consume` 命令虽然可以独立启动，但仍共享横向的 Handler、Service、Repository 和基础设施。当前 Service 层存在文章、评论、点赞、用户和通知之间的直接依赖。

本设计将原先的“一次规划并逐步抽取四个服务”调整为严格的两阶段过程：先完成单体内部 DDD 改良严格分层，确认架构质量后，再抽取三个微服务。阶段边界是硬门槛，不允许在阶段一中提前实施阶段二的服务化改造。

## Goals / Non-Goals

**Goals:**

- 阶段一在当前单体内建立可验证的 DDD 改良严格四层架构。
- 阶段一保持现有 HTTP、开放 gRPC、Kafka 行为和数据库业务结果兼容。
- 阶段一消除 Handler 直接访问 Repository、Domain 依赖框架和业务模块直接跨域访问数据层的问题。
- 阶段二基于阶段一稳定的领域边界抽取三个业务微服务。
- 将通知作为 Community Service 内部能力，而不是独立微服务。
- 支持每个阶段独立测试、评审、确认和回滚。

**Non-Goals:**

- 阶段一不拆分微服务进程、不拆分部署单元、不拆分数据库实例或 Schema。
- 阶段一不改变客户端可见的 HTTP 路径、统一响应格式、错误码和开放 gRPC proto。
- 阶段二不再细分评论、点赞、浏览和通知，不建立第四个业务微服务。
- 第一阶段不引入 Outbox、事务消息或分布式事务框架。
- 本次只更新规划文档，不直接实现代码。

## Decisions

### 1. 两阶段执行并设置人工确认门槛

实施顺序固定为：

```text
阶段一：单体 DDD 改良严格四层架构
        ↓ 测试、架构检查、人工确认
阶段二：抽取三个微服务
```

阶段一的完成条件包括：

- 所有现有业务模块已经归入明确的领域模块。
- 四层依赖规则通过自动化检查。
- 现有单元测试、接口测试和回归测试通过。
- 外部 HTTP/gRPC/Kafka 行为无非预期变化。
- 项目负责人明确确认阶段一完成。

在人工确认之前，阶段二的任务必须保持未执行状态。阶段二的设计可以提前写入文档，但不能提前创建独立服务运行时或切换生产流量。

### 2. 阶段一采用严格的 DDD 四层架构

每个领域模块统一采用以下四层：

```text
Interfaces → Application → Domain
Infrastructure → 实现 Domain/Application 定义的接口
```

#### Interfaces 层

负责协议适配，不承载领域规则：

- HTTP Handler 和路由适配。
- 开放及内部 gRPC Handler。
- Kafka Consumer 和消息反序列化。
- 请求参数校验、用户上下文提取、响应 DTO 转换。
- 将异常和领域错误映射为统一外部错误。

Interfaces 层不得直接调用 GORM、Redis、MongoDB、Kafka Producer 或具体 Repository。

#### Application 层

负责用例编排和应用边界：

- 注册、登录、文章发布、评论、点赞、通知查询等用例。
- 事务边界和用例级权限校验。
- 聚合之间的协作。
- 调用 Domain Service 和 Port。
- 发布需要异步处理的领域事件。

Application 层依赖抽象接口，不依赖具体 GORM Repository、Redis Client、Kafka Client 或 MinIO Client。

#### Domain 层

负责核心业务规则：

- 聚合根、实体和值对象。
- 文章状态流转、评论层级、点赞幂等、通知规则等领域规则。
- 领域服务和领域事件。
- Repository、事件发布、外部查询等 Port 接口。
- 与框架无关的领域错误和不变量。

Domain 层不得导入 Gin、GORM、Redis、MongoDB、Kafka、MinIO 或 HTTP/gRPC 框架。

#### Infrastructure 层

负责具体技术实现：

- GORM/MySQL Repository 实现。
- Redis 缓存和分布式锁适配器。
- MongoDB 通知 Repository。
- Kafka Producer、Consumer 和死信适配器。
- MinIO 对象存储适配器。
- 外部 gRPC Client 和配置加载实现。

Infrastructure 层可以依赖 Domain/Application 定义的接口，但业务规则必须留在 Domain/Application 层。

### 3. 阶段一按三个未来微服务边界组织领域模块

阶段一不创建独立微服务，但目录和依赖边界直接按照未来三个服务设计：

- **Identity 模块**：用户、认证、会话、角色、个人资料和头像。
- **Content 模块**：文章聚合、文章状态、文章查询和文章图片。
- **Community 模块**：评论、回复、文章点赞、评论点赞、浏览、统计、热榜和通知。

通知虽然保留独立的通知能力规格，但实现上属于 Community 模块内部的子模块，阶段二与 Community 一起部署。

模块之间只能通过 Application Port、Domain Event 或明确的内部接口交互，不能通过导入对方 Repository 或直接读写对方模型实现协作。

### 4. 阶段一保持单体运行方式

阶段一继续使用当前单体运行方式：

- `server` 继续提供 HTTP 接口。
- `grpc` 继续提供开放 gRPC 接口。
- `kafka-consume` 继续处理 Kafka 消息。
- MySQL、Redis、MongoDB、Kafka 和 MinIO 连接方式保持兼容。

阶段一可以调整包目录、依赖组装和内部调用方式，但不新增必须独立部署的服务。这样可以先验证领域模型和分层规则，再引入网络调用和服务治理复杂度。

### 5. 阶段二抽取三个微服务

阶段二的服务边界固定为：

1. **Identity Service**：身份与账号。
2. **Content Service**：文章内容。
3. **Community Service**：互动与通知。

统一 HTTP 入口继续保留作为边缘适配层，不计入业务微服务数量。开放 gRPC 的 UserService、ArticleService 和 CommentService 保持兼容，由对应服务实现或由入口适配转发。

### 6. 阶段二的数据所有权

迁移阶段允许共享现有 MySQL，但必须执行逻辑数据所有权：

| 领域模块/服务 | 负责拥有或写入的数据 |
| --- | --- |
| Identity | `users` 及身份会话数据 |
| Content | `articles` 及文章内容资源元数据 |
| Community | `comments`、`article_likes`、`comment_likes`、`article_view_histories`、互动统计和 MongoDB 通知集合 |

Community 负责点赞数、评论数、浏览量和热榜统计。Content 不再被其他服务直接写入统计字段；文章详情通过 Community 的查询接口或统计读模型获得互动数据。

物理数据库拆分属于阶段二后半部分，必须在逻辑所有权稳定、数据校验完成并通过迁移评审后执行。

### 7. 阶段二的同步与异步通信

- gRPC：用户信息查询、文章存在性/发布状态校验、互动统计查询和必要的同步用例调用。
- Kafka：浏览统计、通知生成、统计投影刷新等非关键副作用。

服务间 gRPC 必须使用服务身份、Trace ID、明确超时和错误映射。Kafka 事件必须包含事件 ID、事件类型、版本、发生时间和业务主键，消费者必须支持幂等、有限重试和死信。

第一阶段不引入 Outbox。主业务事务不得依赖通知是否立即写入；如果未来需要数据库提交和事件发布的强一致，再单独提出后续变更。

### 8. 单仓库目录演进

阶段一建议形成以下模块化目录，阶段二再将模块移动到服务目录：

```text
internal/
  interfaces/       # HTTP、开放 gRPC、Kafka 入口适配
  application/      # 用例和应用服务
  domain/
    identity/
    content/
    community/
  infrastructure/  # MySQL、Redis、Mongo、Kafka、MinIO 适配器
  shared/           # 错误、响应、Trace 和公共工具
```

阶段二再演进为：

```text
services/
  identity/
  content/
  community/
shared/
  contracts/
  platform/
```

阶段一不要求一次性移动所有文件；优先建立依赖规则和边界，再按模块迁移代码。

### 9. 迁移、测试和回滚

阶段一采用“模块化替换”方式：先定义端口和领域模型，再迁移用例，最后替换 Handler/Consumer 的依赖组装。每个模块迁移后立即执行单元测试和接口回归测试。

阶段二采用绞杀者模式：统一入口按路由逐步切换到新服务，旧单体实现保留为回滚目标。写请求切换时必须避免无规则双写；发生异常时先停止新服务写入，再将路由切回旧实现。

## Risks / Trade-offs

- [Risk] 阶段一只改架构不改行为，短期内代码量和适配层可能增加 → 以行为兼容和依赖检查作为阶段一验收标准，避免追求一次性大范围重写。
- [Risk] 四层架构被退化为原有 Service/Repository 的目录搬迁 → 每个用例必须经过 Application，领域规则必须进入 Domain，并增加禁止越层依赖的检查。
- [Risk] 阶段一与阶段二边界混淆 → 在 tasks.md 中明确阶段二任务的启动条件，并要求人工确认后才勾选阶段二任务。
- [Risk] 过渡期共享数据库导致跨域访问反弹 → 建立数据所有权矩阵、代码依赖检查和服务账号权限限制。
- [Risk] 阶段二多服务调用增加延迟和部分失败 → 设置 gRPC 超时、明确降级策略，并对公开用户信息和互动统计使用适当缓存/读模型。
- [Risk] 通知与互动合并后模块规模仍然增长 → 在 Community 内部保持评论、互动统计和通知的独立 Domain/Application 模块，禁止重新形成一个巨型 Service。
- [Risk] Kafka 重复或乱序导致重复通知/计数 → 使用事件 ID、幂等处理、重试和死信机制。

## Migration Plan

### Phase 1: DDD 改良严格四层架构

- 固化当前 HTTP/gRPC/Kafka 行为和测试基线。
- 建立 Identity、Content、Community 三个领域模块。
- 建立 Interfaces、Application、Domain、Infrastructure 四层目录和依赖规则。
- 迁移用户、文章、互动与通知用例，消除 Handler 直连 Repository 和跨域数据层访问。
- 将当前横向 Service 拆分为领域用例、领域规则和基础设施适配器。
- 补充领域层、Application 层、接口层和架构依赖测试。
- 执行完整回归测试并生成阶段一验收报告。
- 等待项目负责人确认阶段一完成。

### Phase 2: 三个微服务抽取

只有在阶段一获得明确确认后，才执行以下任务：

- 建立单仓库多服务启动骨架和共享 contracts/platform。
- 抽取 Identity Service，切换用户和认证路由。
- 抽取 Content Service，切换文章和文章图片路由。
- 抽取 Community Service，切换评论、点赞、浏览、热榜和通知路由/消费者。
- 通过 gRPC 和 Kafka 替换阶段一中的单体模块间调用。
- 更新 Docker Compose、Kubernetes 和服务级健康检查。
- 按数据所有权逐步完成逻辑隔离，再评估物理数据库拆分。
- 执行服务级集成测试、契约测试和流量切换验证。

### Rollback Strategy

- 阶段一每个模块迁移保留旧实现的可回退提交点；如果行为回归，回退当前模块而不是回退整个项目。
- 阶段二以路由切换作为发布开关，旧单体实现保留到新服务通过完整验收。
- 发现新服务异常时，先停止新服务写入，再切回旧路由，避免新旧实现同时写入造成数据冲突。
- 物理数据库拆分前必须完成备份、数据校验、迁移演练和回滚评审。
