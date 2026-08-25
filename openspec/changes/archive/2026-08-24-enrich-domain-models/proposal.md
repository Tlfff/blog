## Why

当前五个上下文的边界和依赖已经清晰，但 User、Article、Comment、Notification 的多数业务规则仍由 Application 直接判断或修改字段，领域对象没有充分保护自身状态。需要在不强制所有模型充血、不改变现有接口和操作流程的前提下，把适合的业务不变量与状态转换收敛到 Domain。

## What Changes

- 对 Article、User、Comment 和 Notification 进行选择性充血化；Like 保持当前轻量模型，不为了形式引入完整聚合。
- Article 使用领域构造与行为封装创建、编辑、发布、移入垃圾箱、恢复和彻底删除校验；Application 方法、调用顺序和外部接口保持不变。
- 明确单管理员即唯一文章作者的业务前提，文章操作继续使用当前登录用户 ID，不新增多作者、多管理员或授权功能。
- 修复文章彻底删除遗漏：`ClearArticle` 只有在文章已处于删除状态时才能执行；非删除状态返回现有文章状态错误。
- 恢复文章只执行“已删除 → 草稿”状态转换，不转移或重写文章作者。
- User 增加注册构造、登录记录、资料、手机号、头像和密码变更等领域行为。
- 实施时按“独立业务规则、类型安全、组合语义、按值相等和不可变性”评估值对象，只为确有业务价值的概念引入；优先评估 `ArticleStatus`、`PlainPassword`、`PasswordHash`、`NotificationType`，但不预先强制任何候选必须落地。
- Comment 增加主评论/回复构造、可回复判断、删除授权和热度计算等领域行为。
- Notification 使用领域工厂创建当前已接线的文章点赞通知，并在 Domain 中处理通知类型和自通知规则。
- Repository、事务、Redis、Kafka、MinIO、分页、DTO 和跨上下文 Port 调用继续由 Application/Infrastructure 编排。
- 不新增或删除 HTTP、gRPC、Kafka、数据库和缓存功能，不改变现有 Application/Interfaces 对外方法数量及签名。
- 若实施中发现计划外业务逻辑变更或既有 Bug，暂停实现并先分析确认，不自行修正。

## Capabilities

### New Capabilities

- `article-lifecycle`: 固定单管理员/唯一作者模型下的文章生命周期规则，并增加彻底删除前必须已进入垃圾箱的要求。

### Modified Capabilities

- `modular-ddd-architecture`: 增加选择性充血领域模型的架构要求，明确实体、值对象、Application 和 Port 的职责边界。

## Impact

- 主要影响 `internal/article/domain`、`internal/user/domain`、`internal/comment/domain`、`internal/notification/domain` 及其 Application 调用方式和测试。
- Like 上下文、技术 Port、事务传播、缓存和消息链路保持现状。
- HTTP 路由、请求/响应字段、gRPC 契约、Kafka 消息、数据库结构和 Redis Key 不变。
- 唯一可观察行为调整是：未进入垃圾箱的文章不能被彻底删除。
