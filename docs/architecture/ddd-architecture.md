# 模块化单体 DDD 架构

## 1. 运行形态

项目仍然构建为一个 `blog` 二进制，通过 Cobra 提供四个子命令：

```text
blog server          HTTP 服务与 Cron
blog grpc            对外 gRPC 服务
blog kafka-consume   Kafka Consumer
blog migrate         MySQL 建库建表
```

五个上下文是代码和业务边界，不是五个进程或五套数据库。

## 2. 目录结构

```text
internal/
├── user/
├── article/
├── comment/
├── like/
├── notification/
├── platform/
│   ├── bootstrap/
│   ├── config/
│   ├── database/
│   ├── transaction/
│   ├── kafka/
│   ├── redis/
│   ├── oss/
│   ├── ip/
│   ├── security/
│   ├── cron/
│   └── interfaces/
└── shared/
    └── apperrors/
```

每个业务上下文按实际需要包含：

```text
domain      聚合、状态、领域错误和 Repository Port
app         Command、Query、用例编排和 Application Port
infra       MySQL、MongoDB、Redis、Kafka、MinIO 与本地 ACL Adapter
interfaces  HTTP、gRPC 或 Kafka 协议 Adapter，仅建立真实入口
module.go   显式构造上下文并暴露 Facade 与入口 Adapter
```

## 3. 上下文依赖

```text
User
  ▲
  │ 作者/发送方最小快照
Article ◄──────── Notification
  ▲                  ▲
  │ 评论数/点赞数     │ 文章点赞消息
Comment ◄──── Like ───┘
```

- User 拥有用户、会话、密码修改凭证和头像流程。
- Article 拥有文章、文章统计、图片、浏览历史和热榜。
- Comment 拥有评论、回复、删除规则和根评论回复数。
- Like 拥有文章/评论点赞关系和点赞状态缓存。
- Notification 拥有 MongoDB 通知文档及当前文章点赞通知消费。

跨上下文不导入对方 Infrastructure、Interfaces 或 Repository。调用方依赖最小 Application Facade；复杂展示查询由调用方拥有只读 Read Model JOIN。

## 4. 本地事务传播

`internal/platform/transaction` 使用私有 context key 传递当前 `*gorm.DB` 事务。Application 只依赖：

```go
type TransactionManager interface {
    WithinTransaction(ctx context.Context, callback func(context.Context) error) error
}
```

Infrastructure Repository 通过 `transaction.DB(ctx, fallback)` 取得当前事务。该机制用于保持：

- `article_likes` 与 `articles.like_count` 共同提交或回滚；
- `comment_likes` 与 `comments.like_count` 共同提交或回滚；
- 评论新增/删除、根评论回复数与 `articles.comment_count` 共同提交或回滚。

Redis 和 Kafka 保持在数据库事务提交之后执行，不引入 Outbox 或最终一致投影。

## 5. 查询策略

- Article 详情和 Comment 列表使用调用方拥有的字段最小化只读 JOIN，保持分页、排序和展示字段性能。
- User、Article 的单对象最小快照通过本地 Application Facade 查询。
- 批量用户快照由 User Application 提供，避免 N+1。
- 只读 JOIN 不得返回其他上下文聚合或用于写入。

## 6. Platform 归属

- `platform/security`：PBKDF2、用户 Token、开放 gRPC JWT、HMAC。
- `platform/interfaces/http`：统一响应、错误码映射、校验器和通用中间件。
- `platform/interfaces/grpc`：统一错误映射、Interceptor 和 Server 注册。
- `platform/kafka`：Producer、Consumer、消息契约、重试和 offset 提交。
- `platform/bootstrap`：技术资源和五上下文 Module 组合根。

## 7. 测试与检查

```bash
go test ./...
./scripts/check_architecture.sh
openspec validate refactor-to-ddd --type change --strict --no-interactive
```

架构检查会拒绝技术 SDK 进入 Domain/Application、跨上下文依赖 Infrastructure/Interfaces、旧全局技术分层目录、旧 `pkg` 和旧根 `config` Go 包引用。

## 8. 领域模型充血策略

本项目采用选择性充血模型，详细的值对象选择、领域行为归属和 Domain Service 判断见 `docs/architecture/domain-modeling.md`。
