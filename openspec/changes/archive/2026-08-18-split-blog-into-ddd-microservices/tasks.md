## 1. 阶段一：基线与架构约束

- [x] 1.1 固化现有 HTTP 路由、鉴权分组、统一响应结构、错误码和开放 gRPC proto 的兼容性清单
- [x] 1.2 修复 `ip2region` 测试资源路径依赖，并配置可写的 Go build cache，确保 `go test ./...` 可重复执行
- [x] 1.3 建立 Identity、Content、Community 三个领域模块及其数据所有权矩阵
- [x] 1.4 建立当前模块调用矩阵，标记 Handler 直连 Repository、Service 跨域依赖和基础设施泄漏点
- [x] 1.5 定义 Interfaces、Application、Domain、Infrastructure 四层的依赖方向和禁止依赖规则
- [x] 1.6 增加架构依赖检查，阻止 Domain 依赖 Gin、GORM、Redis、MongoDB、Kafka、MinIO 和传输框架
- [x] 1.7 为现有 HTTP、开放 gRPC 和 Kafka 关键行为增加契约测试基线

## 2. 阶段一：四层架构骨架

- [x] 2.1 在当前单体内建立 Interfaces、Application、Domain、Infrastructure 和 shared 目录
- [x] 2.2 建立 Identity、Content、Community 三个领域模块的目录和包边界
- [x] 2.3 定义统一的领域错误、应用错误、Repository Port、外部资源 Port 和事件发布 Port
- [x] 2.4 将 HTTP Handler、开放 gRPC Handler 和 Kafka Consumer 收敛到 Interfaces 层
- [x] 2.5 将配置、日志、Trace、数据库客户端和第三方客户端初始化收敛到 Infrastructure 层
- [x] 2.6 保持 `server`、`grpc` 和 `kafka-consume` 现有启动方式可用

## 3. 阶段一：Identity 领域模块重构

- [x] 3.1 将用户注册、登录、退出、会话失效和改密重构为 Application 用例（验收：原有注册接口的响应格式和错误码不变）
- [x] 3.2 将用户、角色、会话和密码相关规则重构为 Domain 模型或 Domain Service
- [x] 3.3 定义用户 Repository、Token Session、Redis 和头像对象存储 Port
- [x] 3.4 将 GORM 用户 Repository、Token、Redis 和 MinIO 实现移动到 Infrastructure Adapter
- [x] 3.5 将用户资料、公开资料、手机号、头像和用户基本信息查询迁移到 Identity Application
- [x] 3.6 验证登录、会话、多端退出、管理员权限、改密和头像行为与基线兼容

## 4. 阶段一：Content 领域模块重构

- [x] 4.1 将文章创建、编辑、发布、删除、恢复和彻底删除重构为 Application 用例
- [x] 4.2 将文章状态流转、作者权限和文章可见性规则重构为 Domain 模型
- [x] 4.3 定义文章 Repository、对象存储和互动统计查询 Port
- [x] 4.4 将 GORM 文章 Repository、MinIO 图片能力和查询适配器移动到 Infrastructure
- [x] 4.5 将公开列表、后台列表、文章详情、垃圾箱和开放 ArticleService 查询迁移到 Content Application
- [x] 4.6 验证文章分页、状态过滤、权限校验、图片上传和图片转正行为与基线兼容（且核心文章列表API返回数据一致）

## 5. 阶段一：Community 领域模块重构

- [x] 5.1 将评论、回复、评论列表、用户删除和管理员删除重构为 Application 用例
- [x] 5.2 将文章点赞、评论点赞、取消点赞和点赞状态查询重构为 Application 用例
- [x] 5.3 将评论层级、点赞幂等、权限和统计不变量重构为 Domain 规则
- [x] 5.4 将浏览统计窗口、浏览历史、热榜和统计校准规则重构为 Domain/Application 能力
- [x] 5.5 将通知生成、通知查询、未读状态和通知消费作为 Community 内部通知模块重构
- [x] 5.6 定义评论、点赞、浏览、通知、Redis、Kafka、MongoDB 和文章统计查询 Port
- [x] 5.7 将对应 MySQL/MongoDB Repository、Redis、Kafka、死信和统计适配器移动到 Infrastructure
- [x] 5.8 消除评论和点赞模块直接更新文章表或直接访问用户 Repository 的依赖
- [x] 5.9 验证评论、点赞、浏览、热榜、通知、重复消息和死信行为与基线兼容

## 6. 阶段一：单体集成与确认门槛

- [x] 6.1 将所有现有 HTTP、gRPC 和 Kafka 入口改为调用 Application 用例，不允许入口直接访问 Repository
- [x] 6.2 将当前启动 wiring 改为按领域模块组装 Application 和 Infrastructure 依赖
- [x] 6.3 执行领域层、Application 层、Repository 层和接口契约测试
- [x] 6.4 执行 `go test ./...`、静态检查和架构依赖检查
- [x] 6.5 对比重构前后的关键接口响应、错误码、权限结果和统计结果
- [x] 6.6 输出阶段一验收报告，记录已完成边界、遗留问题和回滚点
- [x] 6.7 移除旧的单体横向 Service 层代码（如 `internal/service`），并将遗留测试改为使用新 Application 用例
- [x] 6.8 **阶段门槛：等待项目负责人确认阶段一完成，未确认前不得开始第 7 节及之后任务**

## 7. 阶段二：三服务骨架与共享契约

- [x] 7.1 在确认阶段一后创建 `services/identity`、`services/content`、`services/community` 服务目录和独立启动入口
- [x] 7.2 创建 `shared/contracts`，集中管理内部 gRPC proto、开放 proto 兼容适配和 Kafka 事件信封
- [x] 7.3 创建 `shared/platform`，抽取配置、日志、Trace、统一错误和基础设施客户端能力
- [x] 7.4 定义服务身份、gRPC 超时、错误映射、Kafka 版本、重试、死信和幂等规则
- [x] 7.5 为三个服务增加独立健康检查、优雅退出和服务级配置

## 8. 阶段二：Identity Service 抽取

- [x] 8.1 将阶段一 Identity 模块移动为 Identity Service，并保留四层依赖结构
- [x] 8.2 实现 Identity Service 的内部用户查询 gRPC 接口
- [x] 8.3 将统一入口的用户注册、登录、资料、改密、头像和退出路由切换到 Identity Service
- [x] 8.4 将 Content 和 Community 对用户数据的直接访问替换为 Identity gRPC Client
- [x] 8.5 验证开放 UserService、会话失效、管理员权限和用户资料接口兼容性
- [x] 8.6 增加 Identity Service 的独立构建、启动、探针和部署配置

## 9. 阶段二：Content Service 抽取

- [x] 9.1 将阶段一 Content 模块移动为 Content Service，并保留四层依赖结构
- [x] 9.2 实现 Content Service 的文章查询、状态校验和互动统计查询接口
- [x] 9.3 将统一入口的文章、垃圾箱和文章图片路由切换到 Content Service
- [x] 9.4 将开放 ArticleService 切换到 Content Service，并验证分页和状态过滤兼容性
- [x] 9.5 通过 Identity gRPC 和 Community 统计接口完成文章详情数据组合
- [x] 9.6 增加 Content Service 的独立构建、启动、探针和部署配置

## 10. 阶段二：Community Service 抽取

- [x] 10.1 将阶段一 Community 模块及内部通知模块移动为 Community Service
- [x] 10.2 实现 Community Service 的评论、点赞、浏览、热榜和互动统计接口
- [x] 10.3 实现 Community Service 的通知查询、未读状态和 Kafka 通知消费者
- [x] 10.4 将统一入口的评论、点赞、热榜和通知路由切换到 Community Service
- [x] 10.5 将开放 CommentService 切换到 Community Service，并验证评论统计兼容性
- [x] 10.6 将浏览统计和通知 Kafka consumer group 切换到 Community Service
- [x] 10.7 验证 Identity/Content gRPC 调用、重复请求、重复消息、超时、重试和死信
- [x] 10.8 增加 Community Service 的独立构建、启动、探针和部署配置

## 11. 阶段二：数据、部署与流量迁移

- [x] 11.1 为 Identity、Content、Community 配置独立的服务配置、Redis Key 前缀和 Kafka consumer group
- [x] 11.2 在共享 MySQL 阶段按数据所有权限制三个服务的 Repository 和数据库访问范围
- [x] 11.3 更新 Docker Compose，使统一入口和三个服务可以独立构建、启动和停止
- [x] 11.4 更新 Kubernetes Deployment、Service、ConfigMap、Secret、探针和资源配置
- [x] 11.5 为各服务增加路由切换开关，并分别验证只读和写请求回滚流程
- [x] 11.6 在逻辑所有权稳定后编写独立 Schema/数据库迁移、校验、备份和回滚脚本
- [x] 11.7 完成服务级集成测试、契约测试、流量切换验证和数据一致性校验

## 12. 阶段二：验收与旧代码清理

- [x] 12.1 验证现有 HTTP 路径、统一响应、鉴权、错误码和开放 gRPC 客户端兼容性
- [x] 12.2 验证服务间 gRPC 的服务身份、Trace ID、超时、错误传播和降级行为
- [x] 12.3 验证 Kafka 重复消息、乱序消息、重试、死信和消费者恢复
- [x] 12.4 验证文章统计、热榜、评论计数、点赞计数和通知的最终一致性与可重建能力
- [x] 12.5 对比新旧实现的关键接口和数据统计，确认迁移期间无数据丢失
- [x] 12.6 移除已切换路由、旧跨域 Repository、兼容 Service 和重复单体 wiring
- [x] 12.7 更新项目文档、运行手册、服务依赖图和故障回滚手册
