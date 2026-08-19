## Why

当前代码把评论、点赞、浏览、排行和通知统一放在 Community 领域中，能够支持现有功能，但五类业务模型、数据所有权和演进方向不够清晰。此次调整希望在不增加微服务数量的前提下，将用户、文章、评论、点赞和通知表达为五个独立的界限上下文，并为未来按上下文独立演进保留边界。

本次只调整领域组织、上下文协作和服务内部模块边界，不改变现有客户端可见的 HTTP/gRPC 契约；通知从互动主流程中独立出来，并通过事件异步处理。

## What Changes

- 将逻辑边界调整为 `User`、`Article`、`Comment`、`Like`、`Notification` 五个界限上下文。
- 保持三个微服务部署边界：`identity` 承载 User，`content` 承载 Article，`community` 承载 Comment、Like、Notification。
- 将现有 `internal/domain/{identity,content,community}` 等组织方式重构为按上下文聚合的目录结构；每个上下文保持 `interfaces`、`application`、`domain`、`infrastructure` 四层边界。
- 将文章点赞和评论点赞统一建模为 Like 上下文的两类目标，但保留目标类型和数据访问 Port 的明确边界。
- 将通知从 Community 互动主流程中抽离为 Notification 上下文，通过版本化、可幂等的事件接收点赞和评论等业务事件。
- 明确跨上下文只通过查询 Port、内部接口或领域/集成事件协作，不共享聚合实体，不跨上下文直接访问 Repository。
- 建立按查询场景使用读模型的原则：热榜、聚合统计和高频列表可使用读模型；核心写入和简单查询不强制引入读写分离。

## Capabilities

### New Capabilities

无。此次变更是对现有能力的上下文边界和实现组织调整。

### Modified Capabilities

- `community-engagement-service`: 将评论、点赞、浏览和排行的责任重新表达为 Comment 与 Like 等上下文的协作，并明确统计查询与跨上下文边界。
- `notification-service`: 将通知提升为独立逻辑上下文，通知主流程与点赞/评论主流程通过异步事件隔离。
- `service-contract-compatibility`: 增加五上下文在三个微服务内的内部协作约束，同时保持现有外部契约兼容。

## Impact

- 影响 `internal` 下的领域、应用、接口和基础设施目录，以及 `services/identity`、`services/content`、`services/community` 的装配代码。
- 需要重新定义上下文内 Port、跨上下文查询接口、事件类型和事件消费者幂等边界。
- 现有 HTTP/gRPC 路由、鉴权语义和主要响应契约保持兼容；内部包路径和装配方式会发生变化。
- 数据库物理表暂不要求拆库；数据所有权按上下文逻辑隔离，后续可独立迁移。
- 不默认引入完整 CQRS。读模型只服务于热榜、统计聚合和高频查询等明确场景，并接受最终一致性。
