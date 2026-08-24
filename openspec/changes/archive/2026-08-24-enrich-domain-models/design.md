## Context

当前模块化单体已经完成上下文、Port、事务和 Adapter 隔离，但领域对象仍大量暴露可直接修改的字段。Application 中存在直接构造聚合、直接修改密码/头像/手机号/登录信息，以及组合 `CanXxx + 状态判断 + 状态赋值` 的情况。

本次以当前已归档的 DDD 架构为基础，只调整业务规则的归属，不改变现有 Application/Interfaces 对外方法、Repository/缓存/消息调用顺序和技术契约。唯一批准的业务行为修正是：彻底删除文章前必须处于删除状态。

系统只有一个管理员账号，该账号同时是唯一文章作者。HTTP 管理路由继续由 `MustAuth → AdminCheckMiddleware` 校验；领域方法仍接收当前用户 ID 并保护文章作者身份，以避免非 HTTP 调用绕过聚合规则。

## Goals / Non-Goals

**Goals:**

- 让 Article、User、Comment 和 Notification 的聚合或值对象保护适合自身的业务不变量。
- 让 Application 回归“加载 → 调用领域行为 → 保存/调用 Port”的编排职责。
- 保持现有方法签名、接口契约、事务边界、IO 顺序、错误码和消息格式。
- 为新增领域行为提供直接单元测试，并保留 Application/Adapter 契约测试。

**Non-Goals:**

- 不追求所有字段值对象化，不新增空洞 Domain Service。
- 不将 Like 强制改造成完整聚合。
- 不把 Repository、事务、Redis、Kafka、MinIO、DTO 或分页逻辑放入实体和值对象。
- 不新增多管理员、多作者、文章授权或其他产品功能。
- 不处理实施中发现但未在本设计批准的其他业务 Bug；发现后暂停并先分析确认。

## Decisions

### 1. 选择性充血，而不是全面对象化

只有满足以下条件的逻辑进入 Domain：

- 明确属于某个实体或值对象；
- 表达业务不变量或状态转换；
- 不需要数据库、缓存、消息或对象存储 IO；
- 可在纯内存单元测试中验证。

分页默认值、DTO 组装、批量查询、事务、计数原子更新和技术错误继续保留在 Application/Infrastructure。Like 当前只有简单幂等关系、缓存和跨上下文统计事务，完整聚合会增加读取和并发成本，因此本次不改。

值对象不按字段类型预先指定，而按以下标准逐项评估：

- 是否具有独立且可复用的业务规则；
- 是否需要避免多个相同基础类型互相传错；
- 是否由多个字段共同表达一个领域概念；
- 是否按值相等而不依赖身份；
- 是否适合创建后不可变、通过整体替换更新；
- 引入后是否减少重复判断，而不是增加大量无意义转换。

初始候选及默认判断：

| 候选 | 默认判断 | 原因 |
| --- | --- | --- |
| `ArticleStatus` | 优先评估 | 有固定取值和生命周期转换规则 |
| `PlainPassword` | 优先评估 | 有现有最少 6 位规则，可区分明文输入 |
| `PasswordHash` | 条件采用 | 能区分明文和哈希，但若只包装字符串且增加 Mapper 成本则不创建 |
| `NotificationType` | 优先评估 | 有固定类型集合且当前只允许文章点赞通知进入创建流程 |
| 手机号、昵称、标签、IP | 默认不采用 | 当前独立规则不足，值对象化收益较低 |
| 各类 ID | 默认不批量采用 | 虽可增强类型安全，但会扩大签名和转换范围，不符合本次最小改造目标 |

实施时可以发现并采用新的合适候选，也可以否决上述候选；但必须在测试或设计说明中记录采用/不采用的理由。

### 2. Article 作为本次最主要的充血聚合

Article Domain 增加或调整以下行为：

```text
NewArticle
EditBy
PublishBy
MoveToTrashBy
RecoverBy
EnsureCanPermanentlyDeleteBy
ReplacePromotedContent（仅更新图片转正后的正文）
```

规则归属：

- 构造函数保证作者、标题、正文和初始状态合法；
- `EditBy` 保护作者身份、删除状态和编辑后的状态；
- `PublishBy` 保护作者身份和删除状态；
- `MoveToTrashBy` 保护作者身份并设置删除状态；
- `RecoverBy` 仅允许作者恢复已删除文章，状态恢复为草稿且不修改作者 ID；
- `EnsureCanPermanentlyDeleteBy` 要求操作者是作者且文章已删除。

为保持当前外部操作顺序：

- 创建仍为“构造合法 Article → Repository Create 获取 ID → MinIO 图片转正 → 必要时回写正文”；
- 编辑仍在 MinIO 调用前完成权限和状态校验，再转正图片、应用新字段并保存；
- 删除、发布、恢复仍为“查询 → 领域校验/状态变化 → Repository”；
- 彻底删除仍为“查询 → 领域校验 → Repository Clear”，只增加已删除状态要求。

当前 `RecoverArticle` 会把 `AuthorID` 重写为操作者 ID。单管理员场景下通常不产生差异，但这属于不必要的所有权转移；充血化后由 `RecoverBy` 保持作者不变。

### 3. User 使用聚合行为，并按评估结果采用值对象

User Domain 增加：

```text
NewUser
RecordLogin
UpdateProfile
ChangePassword
ChangePhone
ChangeAvatar
```

密码相关类型按值对象评估结果决定：`PlainPassword` 有现有最少 6 位规则，优先考虑；`PasswordHash` 只有在能明显降低明文/哈希混用风险且不会产生无意义 Mapper 时才采用。若候选未通过评估，`PasswordHasher` 可以继续使用字符串，但密码规则仍须由 Domain 统一表达。PBKDF2 始终由 Platform Security 实现，加载历史用户不得新增会拒绝旧数据的哈希格式校验。

手机号唯一性需要查询其他用户，仍由 Application 先调用 Repository 判断，再调用 `User.ChangePhone`。Session、一次性凭证和 MinIO 也继续由 Application 调用 Port。

### 4. Comment 封装结构、回复和删除规则

Comment Domain 增加：

```text
NewRootComment
NewReply
EnsureReplyable
DeleteBy
HotValue
```

- 构造函数区分主评论和回复，并设置兼容的初始状态；
- Repository 在事务内锁定根评论后调用 `EnsureReplyable`，继续保证并发安全；
- `DeleteBy` 统一判断已删除、普通用户所有权和管理员删除权限；
- `HotValue` 计算现有的点赞数加回复数。

级联软删除、根评论回复数和文章评论数仍由 Application 事务与 Infrastructure 原子 SQL 完成，不放进 Comment 实体。

### 5. Notification 使用领域工厂，不强制增加状态机

Notification Domain 增加文章点赞通知工厂，输入发送方快照、文章快照和事件时间，负责：

- 仅接受当前已接线的文章点赞类型；
- 自己点赞自己的文章时返回“不创建通知”；
- 设置接收者、发送者快照、内容快照、未读状态和创建时间。

Application 仍按当前顺序查询 User、查询 Article、调用工厂、写入 Repository。批量已读直接由 Repository 更新，不为了充血而逐条加载 Notification 聚合。

`NotificationType` 作为值对象候选评估：若采用，由其验证类型集合和当前可创建类型；若不采用，领域工厂仍必须统一执行相同规则，不允许规则重新散落到 Application。

### 6. 不新增 Domain Service

当前可下沉的规则都能自然归属 Article、User、Comment、Notification 或密码值对象。热度计算可继续作为实体方法或纯领域函数，不需要为了目录完整性新增无状态 Domain Service。

未来只有出现“明确属于领域、但同时涉及多个领域对象且无法归属单一聚合”的复杂规则时，再考虑 Domain Service。

### 7. 保持错误与外部方法兼容

Domain 使用上下文内领域错误表达失败原因，Application 继续映射到现有 `apperrors`，从而保持 HTTP 业务码和 gRPC status 映射。现有 Application 和 Interfaces 方法名、参数和返回值保持不变；允许新增、调整或删除仅供内部使用的 Domain 方法。

## Risks / Trade-offs

- **[新增领域校验可能改变错误出现顺序]** → 保持 Handler 校验和 Application Port 调用顺序，Domain 仅在原规则位置执行等价校验；ClearArticle 例外按批准规则新增状态错误。
- **[公开字段仍可被 Application 直接修改]** → 本次先通过代码迁移和测试禁止直接修改受保护字段；是否进一步私有化字段根据 Mapper 成本另行决定，不强制一次完成。
- **[Article 编辑需要在 MinIO 前校验、MinIO 后赋值]** → 领域提供预校验与应用编辑行为，确保未授权或已删除文章不会先执行对象存储操作。
- **[值对象过多导致转换和签名膨胀]** → 每个候选先记录规则、收益和转换成本；不能减少重复规则或误用风险时保持简单类型。
- **[密码值对象可能拒绝历史哈希]** → 只有通过评估才引入 PasswordHash；采用时使用无新增格式限制的历史重建入口。
- **[单管理员假设未来可能失效]** → 领域方法仍保留作者 ID 校验；未来引入多管理员时需要新 Change 明确授权规则。

## Migration Plan

1. 先建立当前 Application 行为与错误顺序测试，重点固定 Article、User、Comment 和 Notification 写用例。
2. 依次迁移 Article、User、Comment、Notification Domain，每完成一个上下文就修改 Application 调用并执行该上下文测试。
3. Article 阶段单独增加 ClearArticle 状态规则和 RecoverArticle 不转移作者的回归测试。
4. 清理 Application 中已被领域行为替代的直接字段修改和组合判断，确认 Like 与所有 Port 调用未改变。
5. 执行全量测试、架构检查、HTTP/gRPC/Kafka 契约测试和 OpenSpec 校验。

本次无数据库迁移。若某上下文测试失败，可按上下文回退其领域行为与 Application 调整，不影响其他上下文。
