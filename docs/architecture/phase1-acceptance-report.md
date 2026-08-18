# 阶段一验收报告

## 1. 变更范围

本报告覆盖 `split-blog-into-ddd-microservices` 变更的阶段一：在现有单体内完成 Interfaces、Application、Domain、Infrastructure 严格四层架构改造，保持 HTTP、开放 gRPC、Kafka 与数据库业务行为兼容。

## 2. 已完成边界

- 建立四个层次目录与三个领域模块：`internal/interfaces`、`internal/application`、`internal/domain/identity|content|community`、`internal/infrastructure`、`internal/shared`。
- Identity：注册、登录、退出、会话、改密、资料、头像、用户信息查询已迁移为 Application 用例；用户/会话/密码规则进入 Domain；GORM/Redis/MinIO 能力进入 Infrastructure Adapter。
- Content：文章生命周期、分页列表、详情、垃圾箱、开放列表与图片上传/转正已迁移为 Application 用例；状态/权限规则进入 Domain；Repository/MinIO 进入 Infrastructure Adapter。
- Community：评论、点赞、浏览、热榜、通知已迁移为 Application 用例；评论层级、点赞幂等、权限、统计与通知规则进入 Domain；MySQL/MongoDB/Redis/Kafka 进入 Infrastructure Adapter。
- HTTP Handler、开放 gRPC Handler、Kafka Consumer 与 Cron 全部改为调用 Application 接口，不再直接访问 Repository。
- 启动 wiring 按领域模块组装 Application 与 Infrastructure 依赖；`server`、`grpc`、`kafka-consume` 命令保持可用。
- 新增契约测试基线（HTTP 路由、响应结构、错误码、开放 gRPC、Kafka 消息）与领域/应用/仓储层测试。

## 3. 验证结果

| 检查项 | 结果 |
| --- | --- |
| `go test ./...` | 通过 |
| `go vet ./...` | 通过 |
| `make check-arch` | 通过 |
| `go build ./...` | 通过 |
| HTTP 契约测试 | 通过 |
| gRPC 契约测试 | 通过 |
| Kafka 契约测试 | 通过 |
| Identity/Content/Community 应用测试 | 通过 |
| 领域规则测试 | 通过 |
| Repository 层测试 | 通过 |

## 4. 遗留问题

- Kafka 事件仍使用原 JSON 信封，暂未包含事件 ID 与版本字段；重复消息幂等、乱序与死信完善属于阶段二扩展。
- 点赞通知目前仅覆盖文章点赞；评论/回复通知用例尚未接入生产路径。
- 阶段一仍共享 MySQL，Community 通过共享 `articles` 统计列维护点赞/评论/浏览计数；物理隔离属阶段二。
- 旧 `internal/service` 实现保留作为回退对照，阶段二完成流量切换后按任务 12.6 清理。
- `internal/model` 仍被中间件与 DTO 使用，后续可继续向领域模型收敛。

## 5. 回滚点

- 每个领域模块以独立的代码变更集为回退单元：Identity、Content、Community 分别可回退，不影响其他模块。
- `server`、`grpc`、`kafka-consume` 三种启动方式在重构前后均可独立编译；如出现行为回归，优先回退对应模块变更。
- 阶段二开始前建议为阶段一创建 git 提交点，作为后续服务化改造的回滚基线。

## 6. 阶段门槛

阶段一代码与测试已完成，等待项目负责人确认。在获得明确确认之前，任务 7 及之后内容保持未执行状态。
