# 四层依赖规则

## 1. 目标目录结构

```text
internal/
  user/             # 用户界限上下文
  article/          # 文章界限上下文，包含浏览统计与热榜
  comment/          # 评论界限上下文
  like/             # 点赞界限上下文
  notification/     # 通知界限上下文
  shared/           # 跨上下文的纯公共能力
  platform/         # 认证、配置、启动、定时任务与协议适配
```

## 2. 依赖方向

```text
Interfaces -> Application -> Domain
Infrastructure -> Domain/Application 定义的 Port
```

规则：

1. Interfaces 只做协议适配：请求校验、身份提取、DTO 转换、错误映射；不得承载业务规则。
2. Application 编排用例：调用 Domain Service 和 Port，控制事务与应用级权限，发布领域事件。
3. Domain 承载核心规则：聚合、实体、值对象、领域服务、领域事件、Port 接口和领域错误。
4. Infrastructure 实现 Port：GORM/MySQL、Redis、MongoDB、Kafka、MinIO、外部 gRPC Client 等具体适配器。
5. 依赖只允许单向向下；Domain 不得反向依赖 Application/Interfaces/Infrastructure。

## 3. 各层允许依赖

| 层 | 允许依赖 |
| --- | --- |
| Interfaces | Application 接口/用例、共享 DTO、共享错误、协议框架（Gin/gRPC/Kafka SDK） |
| Application | Domain 模型、Domain Service、Port、共享类型 |
| Domain | Go 标准库、Domain 内部类型、共享纯类型 |
| Infrastructure | Domain/Application 定义的 Port、技术框架 |

## 4. 禁止依赖

### Domain 层禁止

- `github.com/gin-gonic/gin`
- `github.com/gin-contrib/*`
- `gorm.io/gorm`、`gorm.io/driver/*`
- `github.com/redis/go-redis/*`
- `go.mongodb.org/mongo-driver/*`
- `github.com/segmentio/kafka-go`
- `github.com/minio/minio-go/*`
- `google.golang.org/grpc`
- `net/http` 及 HTTP/gRPC 传输框架

### Application 层禁止

- 具体 Repository 实现（只允许 Port 接口）
- Redis/Kafka/MongoDB/MinIO 客户端
- GORM `*gorm.DB`、Mongo `*mongo.Database`
- HTTP/gRPC 传输框架

### Interfaces 层禁止

- 具体 Repository 实现
- 直接使用 GORM/Redis/MongoDB/Kafka/MinIO 客户端
- 直接调用 Infrastructure 适配器

### 跨域禁止

- 任何模块不得直接导入其他领域的 Repository 实现或数据表模型来读写数据。
- 跨域协作只能通过 Application Port、Domain Event 或明确的内部接口。

## 5. 执行方式

- `make check-arch` 运行 `scripts/check_architecture.sh`。
- 检查脚本扫描五个上下文的 `domain` 与 `application` Go import，命中禁止列表即失败。
- 阶段一后续每个模块迁移完成时必须通过该检查。
