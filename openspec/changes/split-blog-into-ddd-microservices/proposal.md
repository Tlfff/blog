## Why

当前博客系统是一个约 12K 行代码的 Go 单体应用，HTTP、开放 gRPC、Kafka 消费者和多个业务模块共享同一组 Service、Repository、数据库与基础设施。直接进行微服务拆分会同时引入边界、分层、通信、数据和部署等多个变量，难以在每一步确认架构是否正确。

因此本变更改为两阶段执行：第一阶段只在现有单体内完成 DDD 改良严格四层架构，待阶段一完成并由项目负责人确认后，第二阶段再基于已经稳定的领域边界抽取 3 个微服务。

## What Changes

- 将实施过程明确拆成两个有前置确认关系的阶段。
- **阶段一**：在现有单体内完成严格的 Interfaces、Application、Domain、Infrastructure 四层架构改造，保持现有业务行为和外部接口兼容。
- 阶段一禁止提前进行微服务进程拆分、数据库物理拆分和部署拆分。
- 阶段一完成后必须通过测试、架构依赖检查和人工确认，才能开始阶段二。
- **阶段二**：基于阶段一形成的领域边界，抽取 3 个业务微服务：身份与账号、文章内容、互动与通知。
- 将通知能力并入互动与通知服务，不再把通知单独拆成一个微服务。
- 保留统一 HTTP 入口，兼容现有 HTTP 路径、统一响应结构和开放 gRPC 接口。
- 采用单仓库多服务组织方式，阶段一先完成目录和依赖边界的模块化，阶段二再逐步形成独立服务进程和部署单元。
- 阶段二迁移期间允许共享现有 MySQL，但必须按照领域边界执行数据所有权，禁止跨域直接访问 Repository 或数据表。
- 通过 gRPC 承担同步查询和主流程校验，通过 Kafka 承担浏览统计、通知等非关键异步副作用。

## Capabilities

### New Capabilities

- `identity-service`: 提供身份与账号领域能力；阶段一作为单体内 Identity 模块，阶段二抽取为 Identity Service。
- `content-service`: 提供文章内容领域能力；阶段一作为单体内 Content 模块，阶段二抽取为 Content Service。
- `community-engagement-service`: 提供评论、回复、点赞、浏览、统计和热榜能力；阶段一作为单体内 Community 模块，阶段二与通知能力共同抽取为 Community Service。
- `notification-service`: 提供通知行为契约；阶段一作为 Community 模块内的通知子模块，阶段二不单独部署，归属 Community Service。
- `service-contract-compatibility`: 定义阶段一和阶段二都必须遵守的 HTTP、开放 gRPC、内部 gRPC 与 Kafka 契约。

### Modified Capabilities

无。当前 `openspec/specs/` 中没有已存在的能力规格。

## Impact

- 阶段一主要影响当前 `internal/handler`、`internal/grpc`、`internal/service`、`internal/repository`、`internal/mq`、`internal/routes` 和启动 wiring。
- 阶段一新增严格分层目录、领域模型、用例、端口和基础设施适配器，但不改变现有业务接口。
- 阶段二新增单仓库多服务目录、服务级启动入口、服务间 gRPC 客户端和共享协议契约。
- 阶段二影响 MySQL 表的逻辑所有权、Redis Key 归属、MongoDB 通知集合和 Kafka topic/consumer group 规划。
- 需要补充领域层测试、应用层测试、架构依赖检查、接口契约测试和服务集成测试。
- 本次变更只更新设计与实施计划，不直接实现项目代码。
