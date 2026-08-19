## MODIFIED Requirements

### Requirement: Internal synchronous calls are bounded and authenticated

服务间及同一微服务内跨上下文的同步调用 SHALL 使用受保护的服务身份、明确的超时和可识别的错误映射；调用方不得共享对方聚合实体或直接访问对方 Repository。目标服务不可用时，调用方 SHALL 按接口定义返回失败或降级结果，不得无限等待。

#### Scenario: Internal call times out
- **WHEN** 目标上下文在约定超时时间内没有响应
- **THEN** 调用方返回可识别的超时错误或约定的降级结果，并记录 Trace ID

#### Scenario: Unauthorized internal call is rejected
- **WHEN** 未通过服务身份校验的进程调用内部接口
- **THEN** 目标服务拒绝请求且不执行任何业务写操作

#### Scenario: Context does not share aggregate objects
- **WHEN** 一个上下文需要访问另一个上下文的业务数据
- **THEN** 调用方通过查询契约获得防腐后的数据，不直接导入或修改对方聚合及 Repository 实现

### Requirement: Kafka contracts are versioned and idempotent

跨上下文 Kafka 事件 SHALL 包含事件类型、事件 ID、发生时间、业务主键和版本信息；消费者 SHALL 能够识别重复事件并安全重试。Like、Comment 与 Notification 上下文之间的非核心联动 SHALL 优先使用该事件契约。

#### Scenario: Consumer ignores a replayed event
- **WHEN** 消费者再次收到已经成功处理的事件 ID
- **THEN** 消费者不重复执行业务副作用，并正常提交消费进度

#### Scenario: Unknown event version is isolated
- **WHEN** 消费者收到无法解析或不支持的事件版本
- **THEN** 消息进入可观测的失败/死信流程，不影响其他合法消息的消费
