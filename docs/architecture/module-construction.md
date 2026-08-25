# 上下文 Module 构造约定

## 目的

本文件固定 DDD 重构期间的组合根和上下文 Module 装配约定。当前阶段只建立约定和 Platform 资源骨架，不创建 User、Article、Comment、Like、Notification 的业务 Module。

## Module 约定

每个上下文在迁移时提供显式构造函数，构造函数接收该上下文真正需要的资源、Port 和跨上下文 Facade，并返回该上下文的 Module：

```go
func NewModule(deps Dependencies) (*Module, error)
```

Module 只暴露当前入口实际需要的能力，例如：

```text
Module
├── Application Facade
├── HTTP Adapter（存在 HTTP 入口时）
├── gRPC Adapter（存在 gRPC 入口时）
└── Kafka Adapter（存在 Kafka 入口时）
```

约束如下：

1. Module 构造函数完成本上下文的依赖装配和必要校验。
2. Application 不通过全局变量、字符串 Key 或服务定位器查找依赖。
3. Interfaces 只接收 Module 暴露的 Application 用例或 Adapter。
4. Cross-context Port 由调用方定义，Module 在组合根中注入对应 Facade/ACL Adapter。
5. 不存在的 HTTP、gRPC、Kafka 入口不创建空 Adapter。
6. 迁移阶段允许旧实现暂时继续存在，但同一路由、同一 gRPC Service 或同一 Kafka Topic 只能注册一套运行实现。

## 组合根装配顺序

```text
Cobra 子命令
    ↓
Platform Resources
    ↓
上下文 Module（按迁移进度逐个接入）
    ↓
HTTP / gRPC / Kafka / Cron 注册
    ↓
进程生命周期管理
```

`cmd/server.go`、`cmd/grpc.go` 和 `cmd/kafka_consume.go` 只负责选择资源、创建 Module、注册入口和管理生命周期，不负责直接创建业务 Repository 或业务 Service。
