## Why

当前项目按 `handler / service / repository / model` 技术分层组织，业务边界、跨模块依赖和事务责任集中在少数 Service 与组合入口中，导致用户、文章、评论、点赞和通知之间存在直接 Repository 依赖及跨表写入。需要在不改变任何现有功能和外部行为的前提下，将系统重构为边界明确、可持续演进的模块化单体 DDD 架构。

## What Changes

- 以提交 `70b6558709e587ab1408375d1598b6b9e192e770` 为唯一运行行为基线，将业务代码整理为 User、Article、Comment、Like、Notification 五个限界上下文。
- 在各上下文中按需建立 Domain、Application、Infrastructure、Interfaces 四层，并区分 Domain Port 与 Application Port；不为追求目录对称创建空 Adapter 或无业务意义的模型转换。
- 先建立组合根骨架与模块构造方式；每迁移一个上下文即接入组合根；全部迁移后删除旧装配代码并统一 HTTP、gRPC、Kafka 与 Cron 的入口装配。
- 使用 Application 层定义的本地事务协调 Port 和 Infrastructure 层的 GORM 实现，保留点赞关系与文章计数、评论变更与文章计数等现有跨上下文同库事务原子性。
- 跨上下文写入只能通过调用方定义的最小 Application Port；禁止直接依赖其他上下文的 Repository、Infrastructure Model 或内部实现。
- 跨上下文读取采用混合方案：复杂分页和展示查询使用调用方拥有的专用 Read Model Query Adapter 进行只读 JOIN；存在性、可互动性和单对象最小信息查询使用 Application Query Port 与对方 Application Facade。
- 允许简单 MySQL 持久化对象与 Domain Model 务实复用；仅在 MongoDB 技术类型、复杂 Read Model 或存储结构与领域结构明显不一致时建立 Infrastructure Model 和 Mapper。
- 将 `pkg` 下仅供本项目内部使用的数据库、Kafka、Redis、MinIO 等能力收敛到 `internal/platform`，并按既定归属整理认证、安全、配置、事务、Cron 和协议中间件。
- 仅迁移基线中真实存在且已接线的 HTTP、对外 gRPC、Kafka、Cron、数据库、缓存及对象存储链路；不新增事件、通知类型、消费幂等、目标校验、最终一致投影或其他业务能力。
- 允许重命名内部包、类型、构造函数和方法；外部接口、数据含义、关键副作用顺序和兼容行为保持不变。
- 建立面向重构后架构的测试体系。优先保留或补充关键行为特征测试，迁移期间可删除已被等价替代、与旧结构强耦合的测试，不要求机械搬迁所有旧测试。

## Capabilities

### New Capabilities

- `modular-ddd-architecture`：定义五个限界上下文、四层依赖、Port 所有权、跨上下文事务、只读 Read Model、组合根迁移和外部行为兼容等可验证架构要求。

### Modified Capabilities

无。本变更不修改现有业务能力要求；新增的 capability 仅约束重构后的内部架构和兼容性。

## Impact

- 主要影响 `internal`、`pkg`、`cmd` 下的目录组织、依赖方向、事务抽象、模块构造、入口装配及测试代码。
- HTTP 路由、请求与响应字段、业务错误码和权限行为保持不变。
- 对外 gRPC Service、Method、消息字段、鉴权和错误映射保持不变。
- Kafka Topic、Consumer Group、消息 JSON、生产/消费时机、重试与提交行为保持不变。
- MySQL、MongoDB、Redis 和 MinIO 的既有数据含义、Key、对象路径及业务写入顺序保持不变。
- `feature/DDD` 与提交 `82aa39e` 仅可作为目录和部分领域代码的参考来源，不得整体合并、覆盖或作为行为基线。

