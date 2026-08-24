## ADDED Requirements

### Requirement: 领域模型按业务价值选择性充血
系统 SHALL 将明确属于实体或值对象的业务不变量和状态转换封装到 Domain，并 SHALL 避免为没有独立规则的字段或简单数据结构强制创建值对象、领域服务或聚合。

#### Scenario: 实体保护自身状态
- **WHEN** Article、User 或 Comment 发生状态或业务属性变化
- **THEN** Application 调用领域构造或领域行为完成校验和内存状态变化
- **AND** Application 不直接拼装非法实体或绕过领域行为修改受保护状态

#### Scenario: 无业务价值的模型保持简单
- **WHEN** 一个字段没有独立业务规则，或一个上下文只有简单编排需求
- **THEN** 系统不为了形式上的充血模型新增无意义值对象或领域服务

#### Scenario: Like 保持当前模型
- **WHEN** 本次变更完成
- **THEN** Like 上下文保持现有 Application、Repository、缓存和事务设计
- **AND** 不为 Like 强制新增完整点赞聚合

### Requirement: 实体和值对象不执行外部 IO
实体和值对象 MUST 只处理内存中的业务规则和状态变化，不得直接调用 Repository、事务、Redis、Kafka、MinIO、HTTP 或 gRPC 能力。

#### Scenario: Application 执行写用例
- **WHEN** 一个写用例需要加载、修改并保存聚合
- **THEN** Application 通过 Port 加载聚合
- **AND** 调用聚合的领域行为
- **AND** 再通过 Port 保存并执行缓存、消息或对象存储操作

### Requirement: 值对象必须基于实际领域价值选择
系统 SHALL 只在概念具有独立业务规则、类型安全需求、组合语义、按值相等语义或不可变约束时引入值对象，并 MUST 避免只为包装基础类型而创建值对象。

#### Scenario: 评估值对象候选
- **WHEN** 实施 Article、User、Comment 或 Notification 的充血化改造
- **THEN** 对 `ArticleStatus`、`PlainPassword`、`PasswordHash`、`NotificationType` 及实施中发现的其他候选逐项评估业务规则和收益
- **AND** 只有收益明确且不会引入无意义转换的候选才实现为值对象

#### Scenario: 保持简单字段
- **WHEN** 手机号、昵称、标签、IP、ID 或其他字段仅承担数据传递且没有足够独立规则
- **THEN** 系统保持现有简单类型
- **AND** 不为形式上的充血模型增加包装类型

#### Scenario: 密码候选被采用
- **WHEN** 评估确认明文密码或密码哈希适合值对象
- **THEN** 值对象只表达当前已有的密码规则或明文/哈希语义
- **AND** Application 继续通过 PasswordHasher Port 调用 Platform Security
- **AND** User 实体和值对象不直接执行 PBKDF2

### Requirement: Application 保持用例编排职责
Application SHALL 继续负责 Port 调用、事务、执行顺序、错误映射、DTO 组装和跨上下文协作；领域模型 SHALL 负责自身规则，不接管技术流程。

#### Scenario: 领域行为替代字段直改
- **WHEN** Application 完成 User、Article、Comment 或 Notification 写用例
- **THEN** Application 不直接修改由领域行为保护的字段
- **AND** Repository、缓存、消息和对象存储调用顺序保持当前实现兼容

## MODIFIED Requirements

### Requirement: 重构保持行为基线兼容
系统 MUST 以提交 `70b6558709e587ab1408375d1598b6b9e192e770` 为运行行为基线；参考分支代码不得引入基线中不存在或未接线的产品能力。经本 Change 明确批准的文章彻底删除状态校验属于例外修正。

#### Scenario: HTTP 兼容性验收
- **WHEN** 对比充血化改造前后的 HTTP 接口
- **THEN** 路由、Method、请求字段、响应字段、权限行为和中间件顺序保持兼容
- **AND** 未进入垃圾箱的文章执行彻底删除时返回现有文章状态错误

#### Scenario: gRPC 兼容性验收
- **WHEN** 对比改造前后的对外 gRPC 服务
- **THEN** Service、Method、protobuf 字段、JWT/HMAC 行为和 status error 映射保持兼容

#### Scenario: Kafka 兼容性验收
- **WHEN** 对比改造前后的 Kafka 链路
- **THEN** Topic、Consumer Group、消息 JSON、生产和消费时机、重试及 offset 提交行为保持兼容

#### Scenario: 数据与存储兼容性验收
- **WHEN** 在既有 MySQL、MongoDB、Redis 和 MinIO 数据上运行改造后的系统
- **THEN** 表和集合的业务含义、Redis Key、MinIO 对象路径及既有数据读取方式保持兼容
- **AND** 本次改造不要求业务数据迁移
