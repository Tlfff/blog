## 1. 固定行为基线与迁移清单

- [x] 1.1 从提交 `70b6558709e587ab1408375d1598b6b9e192e770` 提取 HTTP 路由、Method、中间件顺序、请求 DTO、响应 DTO、错误码和权限行为清单
- [x] 1.2 提取对外 gRPC Service、Method、protobuf 字段、JWT/HMAC 拦截器顺序和错误映射清单
- [x] 1.3 提取 Kafka Topic、Consumer Group、消息 JSON、生产时机、消费处理顺序、重试和 offset 提交行为清单
- [x] 1.4 提取 MySQL 表、MongoDB 集合、Redis Key、MinIO 对象路径、Cron 任务及各上下文数据写入所有权清单
- [x] 1.5 记录点赞、评论、浏览历史和通知链路的数据库操作、缓存操作及消息发布顺序，标明必须保持的本地事务边界
- [x] 1.6 建立旧测试到五个上下文及外部契约的覆盖映射，标记可保留、需修复夹具、需由新测试替代和可删除的测试
- [x] 1.7 为每个上下文建立最小 Characterization/Golden Test 计划，确保旧测试仅在等价覆盖就绪后删除

## 2. 建立 Platform、事务和组合根骨架

- [x] 2.1 建立 `internal/platform` 的 bootstrap、config、database、transaction、kafka、redis、oss、security、cron 和 interfaces 基础目录，不创建未使用的业务 Adapter
- [x] 2.2 实现 Platform Resources 的配置加载、MySQL、MongoDB、Redis、Kafka、MinIO 初始化与统一关闭骨架，并保持现有连接参数和失败处理顺序
- [x] 2.3 定义各 Application 可声明的最小 `TransactionManager` Port，并实现共享 GORM TransactionManager
- [x] 2.4 实现私有事务 context 的写入与解析机制，使 Infrastructure Repository 能优先使用当前事务且不向 Application/Domain 暴露 `*gorm.DB`
- [x] 2.5 为 TransactionManager 增加提交、回滚、无事务回退和跨两个 Repository 共享同一事务的测试
- [x] 2.6 确立上下文 Module 构造约定，显式暴露 Application Facade 和真实存在的 HTTP/gRPC/Kafka Adapter，禁止服务定位器
- [x] 2.7 在 `cmd/server.go`、`cmd/grpc.go` 和 `cmd/kafka_consume.go` 接入 Platform Resources 骨架，同时保持旧业务模块继续运行
- [x] 2.8 建立架构检查脚本，禁止 Domain/Application 导入技术 SDK、禁止跨上下文导入 Infrastructure/Interfaces/Repository，并纳入可重复执行命令

## 3. 迁移 User 上下文

- [x] 3.1 为用户注册、登录、退出、公开/个人资料、资料修改、密码流程、头像流程和用户 gRPC 查询补充最小基线测试
- [x] 3.2 建立 User Domain 的用户、角色、状态、Session 语义、领域错误和 Repository Port，保持现有规则与错误文本映射
- [x] 3.3 建立 User Application 的注册登录 Command、资料/账户/密码 Command、公开资料 Query、批量公开资料 Query 及 Result 类型
- [x] 3.4 在 User Application 定义 PasswordHasher、SessionStore、PasswordChangeTokenStore、AvatarStorage 等最小 Port，并迁移现有用例编排顺序
- [x] 3.5 迁移 User MySQL Repository；对一一对应的 MySQL 模型采用 Domain Model 务实复用，不引入无语义 Mapper
- [x] 3.6 在 Platform Security 和 User Infrastructure 中实现 PBKDF2、JWT Session、密码修改凭证、Redis Session 与 MinIO 头像 Adapter
- [x] 3.7 迁移 User HTTP Interfaces，保持 Gin binding、校验顺序、路由、响应和错误码兼容
- [x] 3.8 迁移 User gRPC Interfaces，保持 protobuf、鉴权和 status error 映射兼容
- [x] 3.9 构造 User Module，向其他上下文暴露最小公开资料 Application Facade，并接入 server 与 grpc 组合根
- [x] 3.10 增加 User Domain、Application Port、Repository/Security Adapter 和 HTTP/gRPC 契约测试
- [x] 3.11 在等价测试通过后删除旧 User Service、Repository、Model、DTO、Handler 及与旧构造方式强耦合的测试

## 4. 迁移 Article 上下文

- [x] 4.1 为文章生命周期、状态变化、列表/详情、正文图片、浏览历史、浏览量、热榜和文章 gRPC 查询补充最小基线测试
- [x] 4.2 建立 Article Domain 的文章、状态、浏览历史、热榜语义、领域错误和 Repository Port
- [x] 4.3 建立 Article Application 的创建、编辑、发布、删除、恢复、彻底删除和图片处理 Command
- [x] 4.4 建立 Article Application 的列表、详情、热榜、浏览历史和统计 Query/Result，并定义事务、缓存、对象存储、消息发布和用户查询 Port
- [x] 4.5 提供供 Comment/Like/Notification 使用的最小 Article Application Facade，包括基线真正需要的文章查询和统计调整能力
- [x] 4.6 迁移 Article MySQL Repository、浏览历史 Repository、Redis 热榜/缓存和 MinIO 图片 Adapter
- [x] 4.7 实现 Article 拥有的列表/详情 Read Model Query Adapter，保留现有分页、排序及只读 JOIN 用户展示字段的 SQL 行为
- [x] 4.8 实现 Article 到 User Application Facade 的最小查询 Adapter，并为不依赖 JOIN 的场景支持批量公开资料查询
- [x] 4.9 迁移浏览历史 Kafka Publisher 与 Consumer Application 调用链，保持消息 JSON、发送/消费时机和写入顺序
- [x] 4.10 迁移 Article HTTP、gRPC 和实际存在的 Kafka Interfaces，不创建空 Adapter 包
- [x] 4.11 将热榜初始化和 Cron Job 改为调用 Article Application Facade，保持现有 Cron 表达式和执行行为
- [x] 4.12 构造 Article Module 并接入 server、grpc、kafka-consume 与 Cron 组合根
- [x] 4.13 增加 Article Domain、Application、Read Model JOIN、存储/Kafka Adapter 和 HTTP/gRPC/Kafka 契约测试
- [x] 4.14 在等价测试通过后删除旧 Article Service、Repository、Model、DTO、Handler、浏览历史 MQ 代码及旧测试

## 5. 迁移 Comment 上下文

- [x] 5.1 为主评论、回复、分页查询、删除权限、级联删除、评论计数和评论 gRPC 统计补充最小基线测试
- [x] 5.2 建立 Comment Domain 的评论、回复关系、状态、删除规则、领域错误和 Repository Port
- [x] 5.3 建立 Comment Application 的创建、删除 Command，根评论/回复列表 Query，以及评论统计 Query/Result
- [x] 5.4 在 Comment Application 定义 TransactionManager、Article Query/Counter、用户最小查询和评论缓存等 Port
- [x] 5.5 迁移 Comment MySQL Repository，并保持根评论锁定、级联状态更新和回复计数操作顺序
- [x] 5.6 实现 Comment 拥有的列表 Read Model Query Adapter，保留当前分页、排序以及对评论者/被回复者的只读用户 JOIN
- [x] 5.7 实现 Comment 到 Article Application Facade 的查询与评论数调整 Adapter，并通过事务 context 参加同一个 MySQL 事务
- [x] 5.8 提供供 Like/Notification 使用的最小 Comment Application Facade，不暴露 Comment Repository 或持久化模型
- [x] 5.9 迁移 Comment HTTP 和 gRPC Interfaces，保持路由、权限、DTO、错误码和 protobuf 兼容
- [x] 5.10 构造 Comment Module 并接入 server 与 grpc 组合根
- [x] 5.11 增加 Comment Domain、Application、级联事务回滚、Read Model JOIN 和 HTTP/gRPC 契约测试
- [x] 5.12 在等价测试通过后删除旧 Comment Service、Repository、Model、DTO、Handler 及旧测试

## 6. 迁移 Like 上下文

- [x] 6.1 为文章点赞/取消、评论点赞/取消、重复操作、点赞状态、Redis 缓存降级和文章点赞通知发布补充最小基线测试
- [x] 6.2 建立 Like Domain 的目标类型、点赞关系、状态转换、领域错误和文章/评论点赞 Repository Port
- [x] 6.3 建立 Like Application 的文章/评论点赞与取消 Command、点赞状态 Query 和对应 Result
- [x] 6.4 在 Like Application 定义 TransactionManager、Target Query、Article/Comment Engagement Writer、Like Cache 和 Notification Publisher Port
- [x] 6.5 迁移文章点赞与评论点赞 MySQL Repository，保持现有记录复用、状态更新和幂等返回行为
- [x] 6.6 迁移 Redis 点赞集合、锁、占位值、过期和数据库降级 Adapter，保持现有 Key 与操作顺序
- [x] 6.7 实现 Like 到 Article/Comment Application Facade 的目标查询 Adapter，不额外增加基线不存在的显式校验
- [x] 6.8 实现 Like 到 Article/Comment 统计调整 Facade 的 Adapter，并确保点赞关系与目标计数共享本地 MySQL 事务
- [x] 6.9 迁移基线已接线的文章点赞通知 Kafka Publisher，保持消息字段、异步发送方式和事务后发布时机
- [x] 6.10 迁移 Like HTTP Interfaces，不创建当前不存在的 gRPC 或 Kafka Consumer Adapter
- [x] 6.11 构造 Like Module 并接入 server 组合根
- [x] 6.12 增加 Like Domain、Application、缓存降级、跨上下文事务提交/回滚和 HTTP/Kafka 消息契约测试
- [x] 6.13 在等价测试通过后删除旧 ArticleLike/CommentLike Service、Repository、Model、Handler 及旧测试

## 7. 迁移 Notification 上下文

- [x] 7.1 为通知列表、未读数量、清除未读和当前已接线的文章点赞通知消费补充最小基线测试
- [x] 7.2 建立 Notification Domain 的通知、发送者快照、文章点赞内容、类型、已读状态、领域错误和 Repository Port
- [x] 7.3 建立 Notification Application 的通知查询、清除未读和当前文章点赞通知处理用例及最小查询 Port
- [x] 7.4 建立独立 MongoDB Document Model 和 Mapper，隔离 `primitive.ObjectID`、BSON 与 Domain Model
- [x] 7.5 迁移 Notification MongoDB Repository，保持集合、过滤、分页、排序、更新和计数行为
- [x] 7.6 实现 Notification 到 User/Article Application Facade 的最小查询 Adapter，仅获取当前文章点赞通知所需字段
- [x] 7.7 迁移 Notification HTTP Interfaces，保持现有路由、DTO、分页默认值、响应和错误码
- [x] 7.8 迁移 Notification Kafka Consumer Interface，保持现有消息解析、重试返回和通知创建顺序，不新增消费幂等或其他通知类型
- [x] 7.9 构造 Notification Module 并接入 server 与 kafka-consume 组合根
- [x] 7.10 增加 Notification Domain、Application、MongoDB Adapter、HTTP 和 Kafka Consumer 契约测试
- [x] 7.11 在等价测试通过后删除旧 Notification Service、Repository、Model、DTO、Handler、MQ 代码及旧测试

## 8. 收敛 Platform、共享代码和 `pkg`

- [x] 8.1 将根 `config` 包迁入 `internal/platform/config`，保持配置文件结构、字段、默认行为和所有子命令读取方式
- [x] 8.2 将 `pkg/database` 迁入 Platform Database，并统一由 Platform Resources 创建连接
- [x] 8.3 将 `pkg/kafka` 迁入 Platform Kafka，保持 Producer、Consumer、重试、批量、提交和关闭行为
- [x] 8.4 将 `pkg/oss` 迁入 Platform OSS，并保持 MinIO 预签名、对象路径和公开 URL 行为
- [x] 8.5 将 `pkg/util/redis`、缓存及锁能力迁入 Platform Redis 或所属上下文，保持 Key、Lua/锁和过期行为
- [x] 8.6 将 IP 查询等仅供项目内部使用的剩余 `pkg` 工具迁入合适的 Platform 包，并修正测试资源定位
- [x] 8.7 将 PBKDF2、JWT、HMAC 技术实现收敛到 Platform Security，并将 HTTP/gRPC 鉴权适配器收敛到 Platform Interfaces
- [x] 8.8 拆分现有 `internal/common` 与 `internal/consts`，将领域错误和业务 Key 下沉到所属上下文，将协议响应/错误映射和技术错误移入 Platform
- [x] 8.9 删除已无引用的 `pkg` 路径和临时兼容包，并通过搜索和架构检查确认业务代码不再导入旧路径

## 9. 统一入口与删除旧架构

- [x] 9.1 统一 HTTP 路由装配，由各上下文 Module 提供 Handler，保持所有路由路径、Method、分组和中间件顺序
- [x] 9.2 统一 gRPC Server 装配，由 User、Article、Comment Module 注册现有 Service 并保持 Interceptor 顺序
- [x] 9.3 统一 Kafka Handler 注册，由 Article 和 Notification Module提供当前两个已接线的 Consumer Handler
- [x] 9.4 统一 Cron 注册，由 Platform Cron 调用 Article Application Facade 并保持启动、停止和防重叠行为
- [x] 9.5 精简三个 Cobra 子命令，使其只负责选择资源、构造所需 Module、注册入口和管理生命周期
- [x] 9.6 删除所有短期迁移 Adapter、双重装配和不可达的参考分支代码，确认同一路由/Service/Topic 只有一个实现
- [x] 9.7 删除旧 `internal/service`、`internal/repository`、`internal/model`、`internal/handler`、旧 routes/mq/grpc 装配目录及已被替代测试
- [x] 9.8 检查最终依赖图不存在上下文包循环、跨上下文 Infrastructure 导入和 Application 对技术 SDK 的依赖

## 10. 最终兼容性验收与文档

- [x] 10.1 运行并修复五个上下文的 Domain、Application、Infrastructure 和 Interfaces 测试，确保替代测试覆盖映射完整
- [x] 10.2 执行全量 `go test ./...`，修复或等价替代 Validator 注册、IP 测试资源等基线测试夹具问题
- [x] 10.3 执行架构检查，确认层级依赖、跨上下文依赖、旧目录和旧 `pkg` 引用全部满足设计约束
- [x] 10.4 对比重构前后 HTTP 路由快照、JSON Golden、错误码、权限行为和中间件顺序
- [x] 10.5 对比重构前后 gRPC descriptor、鉴权行为、响应字段和 status error 映射
- [x] 10.6 对比重构前后 Kafka Topic/Group、消息 JSON、生产/消费顺序、重试和 offset 提交行为
- [x] 10.7 验证点赞、评论和浏览历史的事务、缓存及消息副作用顺序，并执行跨上下文失败回滚测试
- [x] 10.8 验证 MySQL/MongoDB Schema 与含义、Redis Key、MinIO 对象路径、配置和部署命令均无迁移要求或行为变化
- [x] 10.9 更新 DDD 架构、目录、模块依赖、事务传播、Read Model JOIN、启动和测试说明文档
- [x] 10.10 完成最终代码审查，确认未从 `feature/DDD` 或 `82aa39e` 引入任何基线不存在的功能后再进入发布验收
- [x] 10.11 逐项验证 `modular-ddd-architecture` spec 的所有 Requirement 和 Scenario，并记录对应自动化检查或测试证据
