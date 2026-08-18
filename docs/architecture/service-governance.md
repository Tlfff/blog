# 服务治理规则

## 1. 服务身份

- 每个服务有固定 `service_id`：`identity-service`、`content-service`、`community-service`。
- 内部 gRPC 调用方通过 metadata `x-service-id` 声明身份，由服务端 `ServerAuthInterceptor` 按允许列表校验。
- 服务端配置通过 `shared/platform/config.Service.Peers` 声明允许调用自己的伙伴。

## 2. gRPC 超时与 Trace

- 内部调用必须设置明确超时，默认 3 秒，可覆盖为 5 秒；不允许无限等待。
- 调用方通过 `x-trace-id` 透传链路 ID；网关未携带时由 `shared/platform/trace` 生成。
- 目标服务不可用时，调用方返回可识别的超时/不可用错误，不阻塞主流程。

## 3. 错误映射

- 服务端通过 `shared/platform/errors.ToGRPC` 将业务错误映射为 gRPC code：
  - 不存在 -> `NotFound`
  - 权限不足 -> `PermissionDenied`
  - 认证失败 -> `Unauthenticated`
  - 参数错误 -> `InvalidArgument`
  - 其他 -> `Internal`
- 客户端通过 `FromGRPC` 把 gRPC code 映射回统一业务错误码，保持 HTTP 错误码兼容。

## 4. Kafka 契约

- 事件必须使用 `shared/contracts/events.Envelope`，包含 `event_id`、`event_type`、`version`、`occurred_at`。
- 消费者必须使用 `event_id` 做幂等：重复事件只应用一次业务影响。
- 未知或不支持的 `version` 进入可观测的失败/死信流程，不影响其他消息消费。
- 每个服务的 Kafka consumer group 相互独立，topic 归属按数据所有权矩阵执行。

## 5. 重试与死信

- 临时错误按 topic 配置重试，默认 3 次，重试间隔 100ms。
- 超过重试上限的消息写入 `dead_letter` topic，由死信消费者观测和处理。
- 主业务事务不依赖通知/统计异步处理结果；异步失败不回滚主流程。
