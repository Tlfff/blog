# 服务依赖图

```text
                            ┌──────────────────────────┐
                            │  统一入口 blog-server       │
                            │  HTTP :8080 / gRPC :9100   │
                            └───────┬───────┬───────┬───┘
                                    │       │       │
                        gRPC(9101)  │       │ gRPC(9102) │ gRPC(9103)
                                    ▼       ▼       ▼
                            ┌─────────┐ ┌─────────┐ ┌─────────────┐
                            │Identity │ │Content  │ │ Community   │
                            │Service  │ │Service  │ │ Service     │
                            └────┬────┘ └────┬────┘ └──────┬──────┘
                                 │          │             │
                                 │ gRPC     │ gRPC        │ gRPC
                                 └──────────┴──────┬──────┘
                                                   ▼
                                             共享 MySQL / MongoDB / Redis / Kafka / MinIO
```

| 调用方 | 被调用方 | 接口 |
| --- | --- | --- |
| 统一入口 | Identity | 认证、会话、资料、头像、用户查询 |
| 统一入口 | Content | 文章生命周期、列表、详情、图片凭证 |
| 统一入口 | Community | 评论、点赞、浏览、热榜、通知 |
| Content | Identity | 作者公开信息 |
| Content | Community | 点赞状态 |
| Community | Identity | 用户公开信息 |
| Community | Content | 文章基本信息 |

## 数据流

- 同步：HTTP/开放 gRPC -> 统一入口 -> 内部 gRPC -> 各服务。
- 异步：统一入口/Community 发布 Kafka 事件，Community Service 消费者处理浏览历史与通知。
- 热榜：Community Service 启动时重建，定时任务由统一入口通过 `RebuildHotRank` 触发。
