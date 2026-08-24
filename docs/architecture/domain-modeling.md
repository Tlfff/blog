# 选择性充血领域模型设计

## 1. 目标

本项目采用选择性充血模型：实体和值对象保护明确属于自身的业务不变量，Application 继续负责 Port 调用、事务、执行顺序、DTO 和跨上下文协作。不会为了形式给所有字段包装类型，也不会创建空洞 Domain Service。

## 2. 值对象评估结果

| 候选 | 结论 | 原因 |
| --- | --- | --- |
| `ArticleStatus` | 采用 | 固定取值并参与发布、删除、恢复和彻底删除状态转换 |
| `PlainPassword` | 采用 | 表达“创建新密码”的明文输入，并集中现有至少 6 位规则 |
| `PasswordHash` | 采用 | 防止明文和哈希字符串混用；历史哈希重建不增加格式限制 |
| `NotificationType` | 采用 | 固定存量类型集合，并区分“合法类型”和“当前已接线可创建类型” |
| 手机号、昵称、标签、IP | 不采用 | 当前没有足够独立规则，包装后主要增加转换成本 |
| UserID、ArticleID 等 ID | 不批量采用 | 类型安全有收益，但会扩大所有 Port 和协议转换范围，不符合本次最小改造目标 |
| Comment 主评论/回复类型 | 不单独采用 | 使用 `NewRootComment`、`NewReply` 构造函数已能表达结构差异 |

## 3. 领域行为归属

### Article

Article 聚合负责：

- 创建时的作者、标题、正文和状态规则；
- 作者编辑；
- 发布；
- 移入垃圾箱；
- 从垃圾箱恢复为草稿；
- 彻底删除前必须已在垃圾箱；
- 图片转正后替换内存正文。

Repository、MinIO 图片移动和失败清理仍由 Application 编排。

### User

User 聚合负责：

- 注册默认角色、状态、头像和时间；
- 记录成功登录；
- 更新资料；
- 修改手机号；
- 修改密码哈希；
- 修改头像对象 Key。

手机号唯一性、PBKDF2、Session、一次性改密凭证和 MinIO 仍由 Application 通过 Port 调用。

### Comment

Comment 聚合负责：

- 主评论和回复构造；
- 已删除主评论不可回复；
- 普通用户只能删除自己的评论；
- 管理员可删除任意评论；
- 评论热度为点赞数加回复数。

级联软删除、行锁、根评论回复数和文章评论数仍由事务和 Repository 处理。

### Notification

Notification Domain 负责：

- 通知类型值的合法集合；
- 当前只有文章点赞通知可进入创建流程；
- 自己点赞自己文章时不创建通知；
- 组装发送方、接收方、文章内容、未读状态和创建时间快照。

User/Article 查询和 MongoDB 写入仍由 Application 调用 Port。

### Like

Like 本次保持现状。其主要复杂度在幂等关系、Redis 缓存和跨上下文事务；强制增加完整聚合会增加查询与并发成本，当前收益不足。

## 4. 未引入 Domain Service 的原因

当前下沉规则都能自然归属 Article、User、Comment、Notification 或值对象。Application 中剩余逻辑主要是 IO 和用例编排，不属于 Domain Service。

只有未来出现“属于领域但无法归属单一实体或值对象”的复杂无状态规则时，才考虑 Domain Service。

## 5. 分层调用方式

```text
Application
    ├── 通过 Port 加载聚合
    ├── 调用实体/值对象的领域行为
    ├── 通过 Port 保存聚合
    └── 按既有顺序处理事务、缓存、消息和对象存储
```

实体和值对象不直接访问 Repository、GORM、Redis、Kafka、MinIO、HTTP 或 gRPC。
