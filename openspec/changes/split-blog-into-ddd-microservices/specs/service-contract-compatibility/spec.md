## Purpose

为两阶段架构演进提供稳定的外部和内部契约，确保阶段一重构及阶段二三个微服务抽取都不要求现有客户端立即迁移，同时让模块/服务间同步调用和异步事件具备明确的身份、错误、超时、版本和幂等约束。

## ADDED Requirements

### Requirement: Existing HTTP contract remains compatible

统一 HTTP 入口 SHALL 保留当前公开、登录、管理员和可选登录路由的路径、鉴权语义、统一响应结构和主要错误码；服务迁移不得要求现有客户端同步修改。

#### Scenario: Existing public route remains callable
- **WHEN** 客户端调用现有公开文章、评论或用户资料路径
- **THEN** 系统通过统一入口返回与拆分前等价的成功或业务错误响应

#### Scenario: Existing authenticated route preserves identity
- **WHEN** 客户端携带当前有效会话凭证调用受保护路径
- **THEN** 统一入口将用户 ID、角色和 Trace ID 传递给目标服务，并保持原有权限结果

### Requirement: Existing open gRPC contract remains compatible

开放 gRPC 服务 SHALL 保持现有 UserService、ArticleService 和 CommentService 的方法名、请求字段、响应字段及调用方鉴权语义；新增能力 SHALL 使用兼容的版本扩展。

#### Scenario: Existing partner call succeeds after extraction
- **WHEN** 二方或三方使用当前 proto 方法请求用户、文章或评论数据
- **THEN** 系统返回与拆分前兼容的 protobuf 响应和错误映射

#### Scenario: Partner authentication remains enforced
- **WHEN** 三方请求缺少有效 HMAC 签名或二方请求缺少有效 JWT
- **THEN** 系统拒绝调用，不因服务拆分而绕过原有鉴权拦截

### Requirement: Internal synchronous calls are bounded and authenticated

服务间 gRPC 调用 SHALL 使用受保护的服务身份、明确的超时和可识别的错误映射；目标服务不可用时，调用方 SHALL 按接口定义返回失败或降级结果，不得无限等待。

#### Scenario: Internal call times out
- **WHEN** 目标服务在约定超时时间内没有响应
- **THEN** 调用方返回可识别的超时错误或约定的降级结果，并记录 Trace ID

#### Scenario: Unauthorized internal call is rejected
- **WHEN** 未通过服务身份校验的进程调用内部接口
- **THEN** 目标服务拒绝请求且不执行任何业务写操作

### Requirement: Kafka contracts are versioned and idempotent

跨服务 Kafka 事件 SHALL 包含事件类型、事件 ID、发生时间、业务主键和版本信息；消费者 SHALL 能够识别重复事件并安全重试。

#### Scenario: Consumer ignores a replayed event
- **WHEN** 消费者再次收到已经成功处理的事件 ID
- **THEN** 消费者不重复执行业务副作用，并正常提交消费进度

#### Scenario: Unknown event version is isolated
- **WHEN** 消费者收到无法解析或不支持的事件版本
- **THEN** 消息进入可观测的失败/死信流程，不影响其他合法消息的消费
