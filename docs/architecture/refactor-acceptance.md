# DDD 重构兼容性验收记录

## 验收基线

- 行为基线：`70b6558709e587ab1408375d1598b6b9e192e770`
- OpenSpec Change：`refactor-to-ddd`
- 产品功能：无新增、无删除
- 数据迁移：无

## 自动化证据

| 验收项 | 自动化证据 |
| --- | --- |
| 五上下文及旧目录删除 | `scripts/check_architecture.sh` |
| Domain/Application 技术隔离 | `scripts/check_architecture.sh` |
| 跨上下文依赖隔离 | `scripts/check_architecture.sh` |
| HTTP 路由契约 | `internal/platform/interfaces/http/routes/contract_test.go` |
| 统一响应和错误码 | `internal/platform/interfaces/http/response/response_test.go` |
| gRPC Service/Method | `internal/platform/interfaces/grpc/server/contract_test.go` |
| JWT/HMAC | `internal/platform/security/*_test.go`、gRPC Interceptor 测试 |
| Kafka 消息字段 | `internal/platform/kafka/message/contract_test.go` |
| Kafka Handler 注册 | `internal/platform/interfaces/mq/contract_test.go` |
| User 用例和 Adapter | `internal/user/**/*_test.go` |
| Article Domain/Application/Read Model | `internal/article/**/*_test.go` |
| Comment 与 Article 同事务 | `internal/comment/infra/transaction_test.go` |
| Like 与目标计数同事务 | `internal/like/infra/transaction_test.go` |
| Notification Document Mapper | `internal/notification/infra/mapper_test.go` |
| Notification Kafka 契约 | `internal/notification/interfaces/kafka/handler_test.go` |
| IP 资源定位 | `internal/platform/ip/ip_test.go` |
| 全量编译和测试 | `go test ./...` |

## Spec 场景结论

- 五个限界上下文已经建立，并由各自 `module.go` 显式构造。
- 写模型归属明确；跨上下文写入通过 Application Facade 和事务 context 完成。
- Domain/Application 不依赖 Gin、GORM、Redis、MongoDB、Kafka、MinIO 或 gRPC SDK。
- Article 详情和 Comment 列表的 Read Model JOIN 只读且由调用方拥有。
- User/Article 最小快照通过本地 Application Facade 查询。
- HTTP、gRPC、Kafka、MySQL、MongoDB、Redis、MinIO 和 Cron 契约保持行为基线。
- 未引入评论事件、额外通知类型、Event Envelope、消费幂等、Outbox、内部 gRPC 或微服务部署。
