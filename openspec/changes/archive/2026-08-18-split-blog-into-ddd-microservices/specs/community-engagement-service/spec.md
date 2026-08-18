## Purpose

为博客系统集中提供评论、点赞、浏览统计、热度排行和通知协作能力；阶段一由单体内 Community 领域模块提供，阶段二由 Community Service 统一承载，使互动数据与文章内容数据解耦并保持幂等和一致的业务结果。

## ADDED Requirements

### Requirement: Comment and reply management

互动社区服务 SHALL 支持文章主评论、楼中楼回复、评论查询、作者过滤、分页和评论删除，并 SHALL 按用户归属及管理员权限执行删除校验。

#### Scenario: User creates a root comment
- **WHEN** 已登录用户针对可评论文章提交合法内容
- **THEN** 系统创建主评论并返回评论标识和展示所需信息

#### Scenario: User creates a reply
- **WHEN** 已登录用户针对存在的主评论提交回复，并指定合法的被回复用户
- **THEN** 系统创建子评论并增加对应主评论的回复统计

#### Scenario: User deletes own comment
- **WHEN** 用户删除自己创建的评论
- **THEN** 系统将评论按现有规则处理为不可见状态，并同步维护相关评论统计

#### Scenario: Non-owner cannot delete another user's comment
- **WHEN** 普通用户尝试删除其他用户的评论
- **THEN** 系统拒绝操作；管理员可按管理权限执行删除

### Requirement: Article and comment likes

互动社区服务 SHALL 支持文章和评论点赞、取消点赞、当前用户点赞状态查询，并 SHALL 对重复请求保持幂等，同时维护互动统计。

#### Scenario: Repeated like is idempotent
- **WHEN** 同一用户对同一文章或评论重复发送点赞请求
- **THEN** 系统最多产生一次有效点赞，重复请求不会重复增加点赞数

#### Scenario: Cancel like updates the effective state
- **WHEN** 已点赞用户取消点赞
- **THEN** 系统将该用户的有效点赞状态设为未点赞，并将对应统计减少一次

#### Scenario: Unauthenticated like is rejected
- **WHEN** 未登录调用方尝试点赞或取消点赞
- **THEN** 系统拒绝请求且不改变点赞状态或统计

### Requirement: View statistics and hot ranking

互动社区服务 SHALL 处理文章浏览统计、登录用户浏览历史和热度排行，并 SHALL 对重复浏览事件及重复消息进行幂等处理。

#### Scenario: Published article view is counted
- **WHEN** 用户或访客在统计窗口内首次浏览已发布文章
- **THEN** 系统按既定窗口规则记录一次有效浏览，并更新可查询的浏览统计

#### Scenario: Duplicate view event is ignored
- **WHEN** 同一统计主体在窗口内重复提交相同文章的浏览事件
- **THEN** 系统不重复增加浏览量

#### Scenario: Hot ranking reflects engagement data
- **WHEN** 调用方查询热榜
- **THEN** 系统返回按约定热度规则排序的可用文章，并排除已经删除的文章

### Requirement: Cross-service interaction statistics

互动社区服务 SHALL 对外提供文章和评论互动统计查询，内容服务不得通过直接写入互动表来修改这些统计；跨服务统计更新 SHALL 具备可重试且不会重复计数的行为。

#### Scenario: Content query obtains current statistics
- **WHEN** 内容查询需要展示文章详情中的点赞、评论或浏览统计
- **THEN** 系统通过约定的服务接口或统计读模型返回当前有效统计

#### Scenario: Replayed statistics update is harmless
- **WHEN** 同一个互动更新事件被重复投递或重复处理
- **THEN** 系统最终只应用一次业务影响，不产生重复计数
