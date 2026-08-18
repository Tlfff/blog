# 阶段二验收报告

## 1. 已完成边界

- 三个独立服务：`services/identity`、`services/content`、`services/community`，各自拥有独立启动入口、服务级配置、健康检查与优雅退出。
- 共享契约：`shared/contracts` 集中管理内部 gRPC proto、Kafka 事件信封；`shared/platform` 提供配置、日志、Trace、统一错误、gRPC 客户端/服务端拦截器。
- 服务间通信：Identity/Content/Community 通过内部 gRPC 同步调用；浏览统计与通知通过 Kafka 异步处理。
- 统一入口：HTTP 与开放 gRPC 全部改为通过内部 gRPC 转发，不再连接 MySQL/MongoDB/Kafka/MinIO。
- 路由开关与回滚：三个服务的 `enabled` 开关 + 回滚文档与旧镜像保留策略。
- 部署：Docker Compose 与 K8s 清单支持统一入口和三个服务独立构建、启动、停止。
- 数据所有权：各服务只初始化自己的 Repository；跨域读取通过 gRPC 只读接口。
- 旧实现清理：`internal/service` 横向 Service 层已删除；生产 wiring 不再使用旧跨域 Repository。

## 2. 验证结果

| 检查项 | 结果 |
| --- | --- |
| `go test ./...` | 通过 |
| `go vet ./...` | 通过 |
| `go build ./...` | 通过 |
| `make check-arch` | 通过 |
| `docker compose config` | 通过 |
| 内部 gRPC 契约生成 | 通过 |
| 服务身份拦截器测试 | 通过 |
| Identity/Content/Community 客户端契约测试 | 通过 |
| Kafka 事件信封测试 | 通过 |

## 3. 遗留问题

- 尚未执行真实端到端运行验证（需要 MySQL/Redis/MongoDB/Kafka/MinIO 环境）；服务健康探针与联调建议在部署环境执行。
- 共享 MySQL 阶段仍保留 `articles` 统计列由 Community 更新；物理拆库需要完成备份、回放校验和迁移演练。
- `cmd/kafka-consume` 保留为兼容入口，生产环境消费者由 Community Service 承载。
- Kafka 事件已引入 `event_id/version` 信封，但现有消息仍兼容旧 JSON；生产切换后建议逐步迁移消费者与死信观测。

## 4. 回滚

- 每个服务独立镜像，可单独回退。
- 统一入口可回退到旧单体镜像 `blog-server`（`server`/`grpc`/`kafka-consume` 命令保留）。
- 写请求回滚先停新服务写入，再切回旧路由，禁止双写。
