## Context

当前仓库已具备 Identity、Content、Community 三个逻辑模块和三个服务脚手架；Community 同时承载评论、点赞、浏览、排行和通知。现有外部 HTTP/gRPC 契约需要保持兼容，领域层必须继续隔离 GORM、Redis、Kafka、MongoDB、MinIO 等技术依赖。

## Goals / Non-Goals

**Goals:**

- 形成五个逻辑界限上下文：`user`、`article`、`comment`、`like`、`notification`。
- 保持三个物理微服务：`identity`、`content`、`community`。
- 让上下文拥有独立领域模型、Port、应用服务和数据访问边界。
- 让点赞自身的幂等与状态转换保持强一致，让通知等联动接受最终一致。
- 仅在热榜、聚合统计和高频列表等场景使用读模型。

**Non-Goals:**

- 本次不增加微服务进程，不拆分数据库，不全面引入 CQRS。
- 不改变现有公开路由、gRPC 方法和主要响应语义。
- 不把 Article、Comment 等聚合实体复制或嵌套到 Like、Notification 上下文。

## Decisions

### 1. 五个上下文，三个服务

逻辑目录按上下文组织，服务入口按部署边界组织：

```text
internal/
├── user/
│   ├── interfaces/{http,grpc}/
│   ├── application/
│   ├── domain/
│   └── infrastructure/{mysql,redis,oss}/
├── article/
│   ├── interfaces/{http,grpc}/
│   ├── application/
│   ├── domain/
│   └── infrastructure/{mysql,oss}/
├── comment/
│   ├── interfaces/{http,grpc}/
│   ├── application/
│   ├── domain/
│   └── infrastructure/mysql/
├── like/
│   ├── interfaces/{http,grpc,kafka}/
│   ├── application/
│   ├── domain/
│   └── infrastructure/{mysql,redis,kafka}/
├── notification/
│   ├── interfaces/{http,kafka}/
│   ├── application/
│   ├── domain/
│   └── infrastructure/{mongo,redis}/
└── shared/
```

`services/identity` 装配 `user`，`services/content` 装配 `article`，`services/community` 装配 `comment`、`like`、`notification`，并使用 Article 上下文提供的 View/Ranking 适配器。目录名可以与服务名不同，但服务装配层不得绕过上下文 Port。

### 2. Like 独立上下文，但不独立部署

Like 聚合只保存 `userID`、`targetType`、`targetID` 和状态；文章与评论由各自上下文拥有。点赞目标验证通过查询 Port 完成，点赞成功后发布 `LikeCreated`、`LikeCanceled` 集成事件。点赞记录的唯一性和状态转换在 Like 自己的事务内强一致；计数、通知和排行通过事件或读模型最终一致。

备选方案是继续把 Like 作为 Community 内部模块。该方案改动更小，但无法清楚表达 Like 同时面向 Article 和 Comment、且拥有独立关系数据的边界，因此本次采用独立逻辑上下文。

### 3. Notification 独立上下文

Notification 只负责通知聚合、通知内容快照、未读状态和查询，不由点赞或评论应用服务直接写通知表。它消费点赞、评论、回复等事件，并以事件 ID 做幂等。通知延迟或失败不得回滚原始业务操作。

### 4. 同步查询与异步事件分工

- 需要立即判断目标是否存在、是否允许互动的请求使用同步查询 Port，并设置超时。
- 点赞数、评论数、热榜和通知等非主事务联动使用事件或专用查询接口。
- 跨上下文不得共享实体、值对象实现或 Repository；需要共享时定义防腐层 DTO/Port。

### 5. 读模型按场景引入

读模型不是简单的“数据库读写分离”，而是为查询场景维护的投影。当前建议：

- 热榜：归属 Article 上下文的独立排行读模型，适合 Redis ZSet/缓存投影。
- 文章互动摘要：当文章详情需要高频同时展示点赞、评论、浏览统计时，可维护统计读模型。
- 通知列表：Notification 自己的查询存储本身就是其读写模型，不必额外复制。
- 普通文章、评论详情和点赞状态：先使用各上下文自己的查询模型，不为所有查询增加投影。

读模型通过可重放事件更新，允许短暂最终一致，并必须支持事件去重和重建。它不改变 Article、Comment、Like 聚合的写入归属。

## Risks / Trade-offs

- [跨上下文调用增加复杂度] → 先在同一进程使用内存/应用层 Adapter，接口保持可替换为 gRPC 或 Kafka。
- [事件延迟导致通知或计数短暂不一致] → 使用事件 ID 幂等、重试、死信和读模型重建。
- [目录迁移造成大范围 import 变化] → 分阶段迁移一个上下文，保留兼容适配器，逐步删除旧包。
- [读模型过度扩张] → 只有存在高频、聚合或排序查询瓶颈时才新增投影，并记录其重建来源。
- [共享 MySQL 掩盖逻辑所有权] → 即使暂不拆库，也限制每个上下文只能通过自己的 Repository/Adapter 写入所属数据。

## Migration Plan

1. 先建立五个上下文的目录、契约和装配边界，不改变路由与表结构。
2. 迁移 User、Article 的现有领域代码，补齐上下文内 Port 和兼容适配器。
3. 从 Community 中迁移 Comment 与 Like，先在同一进程内验证同步查询和事件契约。
4. 将 Notification 聚合、Repository 和消费者迁移到独立上下文，切换为消费 Like/Comment 事件。
5. 迁移热榜与统计读模型，执行重复事件、服务超时、通知延迟和契约兼容测试。
6. 删除旧 Community 聚合入口；若迁移失败，回滚服务装配和路由映射，保留数据库表不变。
