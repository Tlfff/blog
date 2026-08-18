# 阶段二集成验证

## 1. 服务级集成测试

- 内部 gRPC 契约由 `shared/contracts/gen/internalv1` 生成，三个服务共享同一份 proto。
- 统一入口客户端契约测试覆盖 Identity/Content/Community 的字段映射与错误映射：
  - `internal/interfaces/grpc/client/identity_test.go`
  - `internal/interfaces/grpc/client/content_test.go`
  - `internal/interfaces/grpc/client/community_test.go`
- 服务身份校验由 `shared/platform/grpc.ServerAuthInterceptor` 提供，`go test ./...` 覆盖拦截器包。

## 2. 契约测试

- HTTP 路由与统一响应：`internal/interfaces/http/routes/contract_test.go`、`internal/common/contract_test.go`
- 开放 gRPC 服务与方法：`internal/interfaces/grpc/server/contract_test.go`
- Kafka 事件信封：`shared/contracts/events/envelope_test.go`
- 开放 proto 兼容性：`proto/blogopen/v1` 与生成代码未变更。

## 3. 流量切换验证

- 统一入口通过三个服务的 `enabled` 开关控制路由（见 `routing-switch-and-rollback.md`）。
- 先验证只读接口，再验证写接口；异常时先停止写请求再回退镜像。

## 4. 数据一致性校验

- 迁移前使用 `scripts/mysql/backup_and_rollback.sh backup` 备份共享库。
- 切换后对比关键接口返回与关键表计数（`users`、`articles`、`comments`、`article_likes`、`comment_likes`、`article_view_histories`、`notifications`）。
- 物理拆库前必须完成备份、回放校验与迁移演练。
