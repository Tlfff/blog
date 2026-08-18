# DDD 实践

本文档记录博客系统在阶段一“DDD 改良严格四层架构”中实际落地的领域驱动设计：界限上下文、聚合、实体、值对象、领域服务、领域事件、Port 与防腐层。内容与当前代码保持一致，并对尚未完善的演进点单独说明。

## 1. 总览

系统采用严格四层架构：

```text
Interfaces → Application → Domain
Infrastructure → 实现 Domain/Application 定义的 Port
```

业务按未来微服务边界拆成 3 个界限上下文，通知作为 Community 内部子模块：

| 界限上下文 | 目录 | 未来服务 |
| --- | --- | --- |
| Identity（身份与账号） | `internal/domain/identity` | `services/identity` |
| Content（文章内容） | `internal/domain/content` | `services/content` |
| Community（互动与通知） | `internal/domain/community` | `services/community` |

跨层共享内容放在 `internal/shared`，通用 Port 标记放在 `internal/domain/ports`。阶段一不拆分进程，但代码依赖边界已经按三个上下文隔离。

## 2. 界限上下文与上下文映射

```text
                    ┌─────────────────────┐
                    │  统一 HTTP / gRPC 入口 │
                    │  (Interfaces 层)    │
                    └─────────┬───────────┘
                              │
      ┌───────────────┬───────┴───────┬───────────────┐
      ▼               ▼               ▼
┌──────────┐   ┌──────────┐   ┌──────────────┐
│ Identity │◄──│ Content  │◄─ │  Community   │
│          │   │          │   │(含 Notification)│
└──────────┘   └──────────┘   └──────────────┘
      │               │               │
      ▼               ▼               ▼
   GORM/Redis      GORM/MinIO      MySQL/MongoDB/Redis/Kafka
```

上下文之间不直接依赖对方 Repository：

- Content 通过 `ArticleInteractionQuery` 查询 Community 提供的点赞状态。
- Community 通过 `ArticleQuery` 只读查询 Content 的文章信息。
- Community 通过 `UserInfoQuery` 只读查询 Identity 的用户公开信息。
- 阶段一这些查询由 Infrastructure Adapter 实现；阶段二会替换为服务间 gRPC Client，接口本身保持不变。

## 3. Identity 上下文

### 聚合与实体

| 类型 | 名称 | 说明 |
| --- | --- | --- |
| 聚合根 | `User` | 用户账号，包含手机号、昵称、密码哈希、角色、状态、登录信息 |
| 实体 | `Session` | 一次登录会话，包含用户 ID、角色、登录时间、IP、设备 |

### 值对象与领域规则

- 密码哈希：`pbkdf2$迭代次数$盐值$哈希`，由 `HashPassword`/`VerifyPassword` 承载。
- 角色：`RoleUser=1`、`RoleAdmin=2`。
- 用户状态：`StatusNormal=1`、`StatusDeleted=2`。
- 聚合根规则：`IsAdmin()`、`IsNormal()`。

### Port

- `UserRepository`：用户持久化。
- `TokenSession`：登录会话的创建、查询、删除、多端退出。
- `PasswordChangeTokenStore`：一次性改密凭证签发与消费。
- `AvatarObjectStorage`：头像对象存储。

### 领域服务

当前密码哈希/校验以领域包函数形式提供（`identity.HashPassword`/`VerifyPassword`），尚未建模为显式的 `PasswordService` struct；`internal/auth` 已改为委托这些函数。

## 4. Content 上下文

### 聚合与实体

| 类型 | 名称 | 说明 |
| --- | --- | --- |
| 聚合根 | `Article` | 文章，包含作者、标题、正文、标签、状态和统计字段 |
| 查询模型 | `ArticleWithAuthor` | 文章详情所需的文章字段 + 作者昵称/头像/IP |

### 值对象与领域规则

- 文章状态：`Deleted=1`、`Draft=2`、`Published=3`；`All=-2`、`AllExceptDeleted=-1` 用于列表过滤。
- 聚合根规则：
  - `IsDeleted()`、`IsPublished()`、`IsDraft()`、`IsPubliclyVisible()`
  - `CanEdit(userID)`、`CanDelete(userID)`、`CanPublish(userID)`
  - `SoftDelete()`、`Publish()`、`Recover()`

### Port

- `ArticleRepository`：文章 CRUD、游标/偏移分页、按状态计数、带作者查询。
- `ArticleImageStorage`：文章图片上传凭证与转正。
- `ArticleInteractionQuery`：点赞状态只读查询（由 Community 提供）。

### 领域服务

图片转正（`PromoteImages`）目前是 Application 层方法，因为它同时涉及对象存储 Port 和文章持久化编排，未下沉为领域服务。文章状态流转规则已放入聚合方法。

## 5. Community 上下文

Community 是当前最大的上下文，内部包含评论、点赞、浏览统计、热榜与通知五个子域。

### 聚合与实体

| 类型 | 名称 | 说明 |
| --- | --- | --- |
| 聚合根 | `Comment` | 主评论或楼中楼回复，含根评论 ID、被回复人、状态与统计 |
| 实体 | `ArticleLike` | 文章点赞记录（通过 Port 表达，不直接暴露 GORM 模型） |
| 实体 | `CommentLike` | 评论点赞记录 |
| 实体 | `ViewHistory` | 浏览历史流水 |
| 聚合根 | `Notification` | 通知聚合，含接收者、发送方、类型、已读状态、内容 |

### 值对象

- `NotifySender`：通知发送方公开信息（用户 ID、昵称、头像）。
- `LikeArticleContent`：点赞文章通知内容（文章 ID、标题）。
- `HotRankItem`：热榜条目（文章 ID、标题、热度、统计字段）。
- `CommentWithUser`：评论列表所需的作者/被回复者画像。

### 领域规则

- 评论层级：`RootID==0` 为主评论，否则为回复；回复必须存在且未删除的主评论。
- 评论权限：`BelongsTo(userID)`；管理员可覆盖。
- 点赞幂等：`LikeStatusLiked=1`、`LikeStatusCanceled=2`；重复点赞/取消不重复写库。
- 浏览统计：游客只增加计数，登录用户额外记录浏览历史。
- 热榜公式：`热度 = 浏览量 + 点赞数 + 评论数`（`CalcHotScore`）。
- 通知规则：发送者与接收者相同时不产生通知。

### Port

- `CommentRepository`：评论创建、查询、删除，包含计数联动。
- `ArticleLikeRepository` / `CommentLikeRepository`：点赞状态与点赞用户集合。
- `ViewHistoryRepository`：浏览历史与浏览量计数。
- `NotificationRepository`：通知写入、分页查询、未读统计、一键已读。
- `ArticleQuery` / `UserInfoQuery`：跨上下文只读查询（防腐层）。
- `LikeCache`：点赞状态缓存与冷启动重建。
- `LikeCountStore`：评论点赞数读取。
- `HotRankStore`：热榜 ZSet 读写。
- `EventPublisher`：通知与浏览历史异步事件发布。

### 领域事件

| 事件 | 字段 | 用途 |
| --- | --- | --- |
| `NotificationEvent` | 通知类型、发送者 ID、目标 ID、发生时间 | 点赞通知异步处理 |
| `ViewHistoryEvent` | 文章 ID、用户 ID、发生时间 | 浏览历史异步处理 |

事件由 `EventPublisher` 适配到 Kafka，消费端在 `internal/interfaces/mq`。当前事件未包含事件 ID 与版本字段，属于阶段二扩展项。

## 6. 聚合设计总结

| 聚合根 | ID | 不变式 | 跨聚合协作 |
| --- | --- | --- | --- |
| `User` | `ID` | 手机号唯一、正常状态才可登录、改密后其他会话失效 | 会话、改密凭证、头像存储 |
| `Article` | `ID` | 草稿/已删除不可公开；作者或管理员才可管理；图片转正后落正式路径 | 作者画像、点赞状态 |
| `Comment` | `ID` | 回复必须挂在有效主评论；删除主评论级联软删回复；评论计数联动 | 文章计数、用户画像、点赞数 |
| `Notification` | Mongo `ObjectID` | 自触发不通知；接收者只能访问自己的通知 | 文章/用户只读查询 |

## 7. 领域服务现状

当前显式建模的领域服务主要是纯函数：

- `identity.HashPassword` / `identity.VerifyPassword`
- `community.CalcHotScore`

业务用例编排（注册登录、文章生命周期、评论创建、点赞、通知生成、热榜重建）位于 Application 层。后续若出现“跨多个聚合、不适合放在单一聚合上的规则”，可把 `PasswordService`、`NotificationRuleService` 等显式建模为 Domain Service。

## 8. 防腐层（Anti-Corruption Layer）

当前防腐层由 Port + Infrastructure Adapter 组成，分为两类：

### 8.1 基础设施防腐

| 上下文 | 防腐对象 | Adapter |
| --- | --- | --- |
| Identity | GORM/Redis/MinIO | `infrastructure/identity/*` |
| Content | GORM/MinIO | `infrastructure/content/*` |
| Community | MySQL/MongoDB/Redis/Kafka | `infrastructure/community/*` |

Domain 不导入任何技术框架；Application 只依赖 Port。

### 8.2 跨上下文防腐

| 调用方上下文 | 被调用方 | 防腐接口 |
| --- | --- | --- |
| Content | Community | `ArticleInteractionQuery.IsUserLikedArticle` |
| Community | Content | `ArticleQuery.FindByID/GetHotListByIDs/GetTopHotArticles` |
| Community | Identity | `UserInfoQuery.FindUserByID` |

当前 Adapter 内部仍会包装共享单体内的 GORM Repository，这是阶段一的过渡形态；阶段二将用 gRPC Client Adapter 替换，接口签名不变，因此不需要改动 Domain/Application。

## 9. 事务与一致性

- 评论创建/删除、点赞写入等“主流程 + 计数联动”使用 `database.RunTx`/GORM 事务，边界位于 Infrastructure Adapter。
- 阶段一不引入 Outbox、事务消息或分布式事务框架；通知与浏览统计通过 Kafka 异步处理，主流程不等待异步结果。
- 重复点赞/取消点赞由 Application 先查缓存状态再写库，保持幂等。

## 10. 待完善与演进

- Kafka 事件增加事件 ID、事件类型、版本、发生时间与业务主键，消费端做幂等、重试和死信。
- 评论/回复通知用例接入生产路径，目前只实现了点赞文章通知。
- 将 `PasswordService`、通知规则等领域服务显式建模。
- 阶段二以 gRPC Client 作为新的防腐层替换共享单体 Adapter。
- 物理数据库拆分后，`Article` 上的互动统计字段将由 Community 统计读模型替代。
