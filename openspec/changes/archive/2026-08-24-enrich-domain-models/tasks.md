## 1. 固定充血化改造基线

- [x] 1.1 为 Article、User、Comment 和 Notification 的当前写用例补充或确认正常路径、失败路径、错误映射和 Port 调用顺序测试
- [x] 1.2 建立 Application 直接构造聚合、直接修改领域字段和组合业务判断的清单，限定本次下沉范围
- [x] 1.3 确认 Like 上下文代码和行为基线，建立本次不得修改 Like 业务实现的检查点
- [x] 1.4 若发现除 ClearArticle 状态遗漏和 RecoverArticle 作者重写外的逻辑 Bug 或行为差异，暂停实现并先提交分析，不自行修正
- [x] 1.5 按业务规则、类型安全、组合语义、不可变性和转换成本评估值对象候选，记录每个候选采用或不采用的理由

## 2. 充血化 Article 聚合

- [x] 2.1 为文章构造、编辑、发布、移入垃圾箱、恢复和彻底删除规则增加 Domain 单元测试
- [x] 2.2 评估并按需引入 ArticleStatus 值对象或强类型，集中固定取值和状态转换规则；若不采用则记录原因
- [x] 2.3 增加 Article 领域构造函数，复用现有标题、正文、状态和作者错误语义
- [x] 2.4 将编辑权限、删除状态和字段更新封装为 Article 领域行为，确保业务校验发生在 MinIO 操作之前
- [x] 2.5 将发布、移入垃圾箱和恢复封装为 Article 状态转换方法，并保证恢复时不修改 AuthorID
- [x] 2.6 增加彻底删除领域校验，要求操作者为作者且文章已处于删除状态，失败映射为现有文章状态错误
- [x] 2.7 修改 Article Application 使用领域构造和行为，保持现有公开方法签名及 Repository、图片转正、失败清理和保存顺序
- [x] 2.8 调整 Article Repository 调用方式以保存聚合状态，且不改变 SQL 字段含义、事务或物理删除行为
- [x] 2.9 增加 ClearArticle 非删除状态拒绝、垃圾箱文章成功清除和 RecoverArticle 作者不变的回归测试
- [x] 2.10 运行 Article Domain、Application、Infrastructure 和 HTTP/gRPC 测试并确认契约兼容

## 3. 充血化 User 聚合和值对象

- [x] 3.1 为用户注册默认状态、登录记录、资料修改、手机号修改、头像修改和密码修改增加 Domain 单元测试
- [x] 3.2 评估 PlainPassword 和 PasswordHash 值对象；只实现通过业务价值评估的候选，并记录未采用候选的原因
- [x] 3.3 若采用 PlainPassword，只封装现有至少 6 位规则并返回现有密码长度错误
- [x] 3.4 若采用 PasswordHash，提供不新增格式限制的历史哈希重建入口；否则保持字符串存储语义
- [x] 3.5 按值对象评估结果调整 PasswordHasher Port 和 Platform Security Adapter，保持 PBKDF2 存储格式不变
- [x] 3.6 增加 User 领域构造函数，封装普通用户、正常状态、默认头像及创建时间等现有初始化规则
- [x] 3.7 增加 RecordLogin、UpdateProfile、ChangePassword、ChangePhone 和 ChangeAvatar 领域行为
- [x] 3.8 修改 User Application 调用领域行为，保持账号唯一性查询、密码校验、Session、一次性凭证、MinIO 和 Repository 调用顺序
- [x] 3.9 按值对象采用结果调整 User Infrastructure Mapper，并保持 users 表结构和字段语义兼容
- [x] 3.10 运行 User Domain、Application、Infrastructure、HTTP/gRPC 和 Platform Security 测试

## 4. 充血化 Comment 聚合

- [x] 4.1 为主评论构造、回复构造、可回复判断、删除授权和热度计算增加 Domain 单元测试
- [x] 4.2 增加 NewRootComment 和 NewReply 领域构造函数，保持当前字段、初始状态和允许的回复结构
- [x] 4.3 增加 EnsureReplyable 领域行为，并在 Repository 事务锁定根评论后调用以保持并发和操作顺序
- [x] 4.4 增加 DeleteBy 领域行为，统一已删除、普通用户所有权和管理员删除规则
- [x] 4.5 增加 HotValue 领域行为，保持点赞数加回复数的现有公式
- [x] 4.6 修改 Comment Application 使用领域构造和行为，保持事务、级联软删除、根评论计数和文章评论数调用顺序
- [x] 4.7 运行 Comment Domain、Application、Infrastructure 和 HTTP/gRPC 测试，包含跨上下文事务回滚测试

## 5. 使用 Notification 领域工厂

- [x] 5.1 为文章点赞通知创建、自通知跳过和未接线通知类型拒绝增加 Domain 单元测试
- [x] 5.2 评估 NotificationType 值对象；若采用则集中类型集合和可创建类型规则，若不采用则记录原因
- [x] 5.3 增加文章点赞通知领域工厂，封装接收者、发送者快照、内容快照、未读状态和创建时间
- [x] 5.4 修改 Notification Application 保持“查询 User → 查询 Article → 调用工厂 → Repository Insert”的现有顺序
- [x] 5.5 保持批量已读、未读数量和列表分页为 Application/Repository 行为，不强制逐条加载聚合
- [x] 5.6 运行 Notification Domain、Application、MongoDB Mapper、HTTP 和 Kafka Consumer 测试

## 6. 清理与最终验收

- [x] 6.1 删除被新领域行为替代的 Application 直接字段修改和重复业务判断，不删除任何现有产品用例或协议方法
- [x] 6.2 检查实体和值对象不导入或调用 Repository、事务、Redis、Kafka、MinIO、HTTP 或 gRPC 能力
- [x] 6.3 检查 Like 上下文业务代码未被充血化改造改变，消息、缓存和事务行为保持原样
- [x] 6.4 对比 HTTP 路由、请求响应、错误码、gRPC、Kafka、数据库和 Redis 契约，除 ClearArticle 状态校验外均保持兼容
- [x] 6.5 更新领域模型设计文档，记录全部值对象候选的采用/不采用结论、选择性充血边界和未引入 Domain Service 的原因
- [x] 6.6 执行 `gofmt`、`go test ./...`、`go build ./...`、`scripts/check_architecture.sh` 和 OpenSpec 严格校验
- [x] 6.7 自检 Git Diff，确认没有新增产品功能、计划外 Bug 修复、无意义值对象、空 Domain Service 或 Like 改造
