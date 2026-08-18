# 运行手册与故障回滚

## 1. 本地启动

```bash
# 基础设施（MySQL/Redis/MongoDB/Kafka）
docker compose up -d mysql redis mongodb kafka

# 三个服务
go run ./services/identity
go run ./services/content
go run ./services/community

# 统一入口
go run . server -p 8080
go run . grpc
```

## 2. 健康检查

三个服务均通过 gRPC health 暴露就绪状态；K8s 使用 TCP 探针检查 9101/9102/9103。

## 3. 常见故障与处理

| 故障 | 现象 | 处理 |
| --- | --- | --- |
| Identity 不可用 | 登录/注册/资料失败 | 检查 9101 健康；确认 Redis/MySQL；回退统一入口镜像 |
| Content 不可用 | 文章列表/详情失败 | 检查 9102；确认 MySQL/MinIO；回退网关 |
| Community 不可用 | 评论/点赞/通知失败 | 检查 9103；确认 MySQL/MongoDB/Redis/Kafka；回退网关 |
| Kafka 堆积 | 通知/浏览延迟 | 检查消费者日志与死信 topic；必要时扩容 Community 副本 |
| 数据不一致 | 迁移期间计数异常 | 停止写请求，使用 `backup_and_rollback.sh` 校验，回退服务镜像 |

## 4. 回滚

- 将 `services/*/config.yaml` 的 `enabled` 改为 `false`，或回退统一入口镜像。
- 旧单体镜像 `blog-server` 的 `server`/`grpc`/`kafka-consume` 命令仍可独立启动。
- 写请求切换期间禁止新老实现同时写同一数据；回滚先停新服务写请求，再切旧路由。
