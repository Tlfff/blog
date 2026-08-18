# shared/contracts

集中管理阶段二的服务间契约。

## 1. 内部 gRPC proto

- 源文件：`shared/contracts/proto/internal/v1/`
- 生成代码：`shared/contracts/gen/internalv1/`
- 服务：
  - `IdentityService`：认证、会话、资料、头像、用户查询
  - `ContentService`：文章生命周期、列表、详情、图片凭证
  - `CommunityService`：评论、点赞、浏览、热榜、通知

## 2. 开放 proto 兼容适配

开放 gRPC proto 仍由 `proto/blogopen/v1` 定义，生成代码位于根目录 `gen/`。
内部 gRPC 与开放 gRPC 独立演进；阶段二由统一入口或各服务提供开放接口适配层，保持现有方法名、请求字段、响应字段与鉴权语义不变。

## 3. Kafka 事件信封

`shared/contracts/events` 定义统一事件信封：

```json
{
  "event_id": "uuid",
  "event_type": "notification",
  "version": "v1",
  "occurred_at": "RFC3339",
  "payload": {}
}
```

事件类型当前包含 `notification` 与 `view_history`。消费者应使用 `event_id` 做幂等，未知版本进入死信流程。
