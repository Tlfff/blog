## Purpose

定义博客单体完成 DDD 重构后必须满足的模块边界、依赖约束、事务一致性、读取协作及兼容性要求，使内部架构可以被自动检查和行为测试验证，同时保证现有产品功能与外部契约不发生变化。

## ADDED Requirements

### Requirement: 系统按五个限界上下文组织业务能力
系统 SHALL 将业务能力组织为 User、Article、Comment、Like 和 Notification 五个限界上下文，并 SHALL 为每个上下文明确其写模型和业务规则的所有权。

#### Scenario: 业务代码完成上下文归属
- **WHEN** DDD 重构完成并检查业务包结构
- **THEN** 用户、文章、评论、点赞和通知业务代码分别归属于对应的五个上下文
- **AND** 不存在继续承载多个上下文业务规则的全局 Service、Repository 或 Model 包

#### Scenario: 上下文拥有自己的写模型
- **WHEN** 一个用例需要修改用户、文章、评论、点赞关系或通知数据
- **THEN** 对应数据只能由拥有该写模型的上下文执行修改
- **AND** 其他上下文不得直接写入该上下文拥有的表、集合或缓存关系

### Requirement: 上下文内部遵守四层依赖方向
系统 SHALL 在每个上下文中按实际需要组织 Domain、Application、Infrastructure 和 Interfaces，并 MUST 保持 `Interfaces → Application → Domain` 的业务依赖方向，由 Infrastructure 实现 Domain 或 Application 声明的 Port。

#### Scenario: Domain 与技术框架隔离
- **WHEN** 检查任一上下文的 Domain 包依赖
- **THEN** Domain 不依赖 HTTP、gRPC、ORM、Redis、MongoDB、Kafka 或对象存储 SDK
- **AND** Domain 不依赖 Application、Infrastructure 或 Interfaces

#### Scenario: Application 与技术实现隔离
- **WHEN** 检查任一上下文的 Application 包依赖
- **THEN** Application 只通过 Port 使用持久化、事务、缓存、消息、对象存储和跨上下文能力
- **AND** Application 不直接持有或创建具体技术客户端

#### Scenario: Interfaces 仅适配外部协议
- **WHEN** HTTP、gRPC 或 Kafka 输入进入一个上下文
- **THEN** Interfaces 完成参数转换、入口鉴权、校验和错误映射后调用 Application
- **AND** Interfaces 不直接访问业务 Repository 或编写领域规则

### Requirement: Port 按领域规则和应用编排分层
系统 SHALL 区分 Domain Port 与 Application Port；领域模型真正需要的持久化抽象归 Domain，用例编排所需的事务、查询、缓存、消息、存储及跨上下文能力归 Application。

#### Scenario: 用例使用外部能力
- **WHEN** 一个应用用例需要事务、缓存、Kafka、MinIO、Session Store 或其他上下文信息
- **THEN** 调用方 Application 声明满足自身最小需求的 Port
- **AND** Domain 不感知这些用例级技术或协作能力

#### Scenario: 跨上下文调用遵循调用方接口
- **WHEN** Like、Comment、Article 或 Notification 需要另一个上下文提供能力
- **THEN** 调用方通过自己定义的最小 Port 调用对方 Application Facade
- **AND** 调用方不依赖对方的 Repository、Infrastructure Model、Interfaces 或 Application 内部实现

### Requirement: 跨上下文写入保持现有本地事务原子性
系统 MUST 保持行为基线中点赞关系与目标统计、评论变更与文章评论统计之间的本地 MySQL 事务原子性，同时 SHALL 避免 Application 和 Domain 依赖具体事务框架。

#### Scenario: 文章点赞事务成功
- **WHEN** 用户执行一次需要写入的文章点赞或取消点赞操作
- **THEN** 点赞关系和文章点赞计数在同一个本地事务中共同提交
- **AND** Redis 更新及现有通知消息发布只在数据库事务成功后按基线顺序执行

#### Scenario: 文章点赞事务失败
- **WHEN** 点赞关系或文章点赞计数中的任一数据库操作失败
- **THEN** 同一事务中的全部数据库修改均回滚
- **AND** 不执行事务成功后才应发生的缓存更新或消息发布

#### Scenario: 评论事务保持原子性
- **WHEN** 创建、删除主评论或删除回复需要同步调整评论计数和文章评论计数
- **THEN** 基线中属于同一事务的数据库操作继续共同提交或共同回滚
- **AND** 调用方通过 Application Port 协作而不是直接使用 Article Repository

### Requirement: 跨上下文读取采用受约束的混合模式
系统 SHALL 根据查询性质使用调用方拥有的只读 Read Model Query 或对方 Application Query Facade，并 MUST 防止跨上下文读取演变为数据写入或 Repository 共享。

#### Scenario: 复杂分页展示查询
- **WHEN** 文章列表或评论列表需要跨上下文展示字段并依赖数据库分页、排序或过滤
- **THEN** 调用方拥有的 Read Model Query 可以执行字段最小化的只读 JOIN
- **AND** 查询只返回调用方定义的 Read Model，不返回或复用其他上下文的聚合与持久化模型

#### Scenario: 规则校验或最小对象查询
- **WHEN** 一个上下文需要存在性、可互动性或单对象最小展示信息
- **THEN** 调用方通过 Application Query Port 调用数据所有者提供的 Facade
- **AND** 批量组装场景使用批量查询能力而不是逐条产生 N+1 查询

### Requirement: 持久化模型分离必须具有实际语义
系统 SHALL 避免为字段完全一一对应的简单 MySQL 对象创建无业务意义的重复模型和 Mapper，但 MUST 隔离会向 Domain 泄露技术类型或明显不同存储结构的持久化表示。

#### Scenario: 简单 MySQL 映射
- **WHEN** MySQL 字段与领域对象的状态一一对应且不存在技术类型泄露
- **THEN** 领域对象可以承担该简单持久化映射
- **AND** Domain 仍不得包含数据库连接、ORM API 或查询表达式

#### Scenario: 技术存储结构需要隔离
- **WHEN** MongoDB 文档包含 ObjectID、BSON 动态结构，或查询结果属于复杂 JOIN/统计投影
- **THEN** Infrastructure 使用独立 Document 或 Read Model
- **AND** 在 Infrastructure 边界完成与 Domain/Application 类型的转换

### Requirement: Adapter 与模块构造按实际运行能力建立
系统 SHALL 只为上下文当前真实存在的 HTTP、gRPC、Kafka 或 Cron 能力建立 Adapter，并 SHALL 通过显式模块构造和组合根完成依赖装配。

#### Scenario: 上下文不存在某种入口
- **WHEN** 一个上下文在行为基线中没有 gRPC Service 或 Kafka Consumer
- **THEN** 重构不得为了目录对称新增对应运行入口或空业务 Adapter

#### Scenario: 子命令装配所需模块
- **WHEN** 启动 `server`、`grpc` 或 `kafka-consume` 子命令
- **THEN** 组合根只初始化该子命令需要的技术资源和上下文模块
- **AND** 业务代码不通过全局服务定位器动态查找依赖

### Requirement: 安全技术与用户业务语义分离
系统 SHALL 由 User 上下文拥有密码规则、Session 生命周期、密码修改凭证、角色和状态语义，并 SHALL 由 Platform 提供 PBKDF2、JWT、HMAC、Redis Session 以及协议鉴权的技术实现。

#### Scenario: 用户应用执行认证用例
- **WHEN** User Application 需要密码哈希、Token Session 或密码修改凭证能力
- **THEN** User Application 通过自己声明的 Port 使用 Platform 或 Infrastructure Adapter
- **AND** Platform 技术实现不拥有或改写用户领域规则

#### Scenario: 协议入口执行鉴权
- **WHEN** HTTP 或 gRPC 请求需要 JWT、HMAC 或角色鉴权
- **THEN** Platform Interfaces 在协议边界完成凭证解析和拦截
- **AND** 现有鉴权结果、错误映射及拦截顺序保持兼容

### Requirement: 重构保持行为基线兼容
系统 MUST 以提交 `70b6558709e587ab1408375d1598b6b9e192e770` 为唯一运行行为基线；参考分支代码不得引入基线中不存在或未接线的产品能力。

#### Scenario: HTTP 兼容性验收
- **WHEN** 对比重构前后的 HTTP 接口
- **THEN** 路由、Method、请求字段、响应字段、错误码、权限行为和中间件顺序保持兼容

#### Scenario: gRPC 兼容性验收
- **WHEN** 对比重构前后的对外 gRPC 服务
- **THEN** Service、Method、protobuf 字段、JWT/HMAC 行为和 status error 映射保持兼容

#### Scenario: Kafka 兼容性验收
- **WHEN** 对比重构前后的 Kafka 链路
- **THEN** Topic、Consumer Group、消息 JSON、生产和消费时机、重试及 offset 提交行为保持兼容

#### Scenario: 数据与存储兼容性验收
- **WHEN** 在既有 MySQL、MongoDB、Redis 和 MinIO 数据上运行重构后的系统
- **THEN** 表和集合的业务含义、Redis Key、MinIO 对象路径及既有数据读取方式保持兼容
- **AND** 本次重构不要求业务数据迁移

### Requirement: 重构不得补齐或新增未接线功能
系统 MUST 只迁移行为基线中真实存在且可达的能力，不得根据未使用常量、不可达方法或参考分支实现补齐功能。

#### Scenario: 参考代码包含额外能力
- **WHEN** `feature/DDD`、`82aa39e` 或旧代码中的不可达路径包含基线未接线的事件、通知、校验或微服务能力
- **THEN** 这些能力不进入本次重构实现

#### Scenario: Kafka 和通知能力迁移
- **WHEN** 迁移评论、点赞和通知相关代码
- **THEN** 不新增评论事件、评论/回复/评论点赞通知、event envelope、消费幂等、Outbox 或最终一致投影
- **AND** 只保留当前已接线的消息链路

### Requirement: 架构和兼容要求必须可验证
系统 SHALL 提供自动化检查或测试，验证层级依赖、上下文隔离、事务原子性和外部契约兼容性。

#### Scenario: 上下文迁移完成
- **WHEN** 任一上下文切换到新模块并准备删除旧实现
- **THEN** 该上下文的关键行为特征测试、Domain/Application 测试、Adapter 测试和架构检查均通过
- **AND** 旧测试仅在已有等价覆盖时删除

#### Scenario: 全部重构完成
- **WHEN** 五个上下文和 Platform 全部迁移完成
- **THEN** 全量测试、架构依赖检查、HTTP/gRPC/Kafka 契约对比和跨上下文事务回滚测试均通过
- **AND** 旧全局技术分层业务目录和旧内部 `pkg` 技术路径不再被业务代码依赖
