## MODIFIED Requirements

### Requirement: Article and comment likes

Like 上下文 SHALL 支持文章和评论点赞、取消点赞、当前用户点赞状态查询，并 SHALL 对重复请求保持幂等，同时维护可查询的互动统计。Like 上下文 SHALL 只持有目标类型与目标标识，不得直接持有或修改 Article、Comment 聚合。

#### Scenario: Repeated like is idempotent
- **WHEN** 同一用户对同一文章或评论重复发送点赞请求
- **THEN** 系统最多产生一次有效点赞，重复请求不会重复增加点赞数

#### Scenario: Cancel like updates the effective state
- **WHEN** 已点赞用户取消点赞
- **THEN** 系统将该用户的有效点赞状态设为未点赞，并将对应统计减少一次

#### Scenario: Like target is validated through a context boundary
- **WHEN** 用户对文章或评论发起点赞，且目标不存在、不可见或不允许互动
- **THEN** 系统拒绝点赞且不写入有效点赞记录

#### Scenario: Unauthenticated like is rejected
- **WHEN** 未登录调用方尝试点赞或取消点赞
- **THEN** 系统拒绝请求且不改变点赞状态或统计

### Requirement: Cross-service interaction statistics

互动能力 SHALL 对外提供文章和评论互动统计查询，内容服务不得通过直接写入点赞或评论数据来修改统计。点赞、评论与通知之间的非核心联动 SHALL 通过可重试且幂等的事件或查询接口完成；点赞记录自身的唯一性和状态转换 SHALL 保持强一致。

#### Scenario: Content query obtains current statistics
- **WHEN** 内容查询需要展示文章详情中的点赞、评论或浏览统计
- **THEN** 系统通过约定的查询接口或统计读模型返回当前有效统计

#### Scenario: Replayed statistics update is harmless
- **WHEN** 同一个互动更新事件被重复投递或重复处理
- **THEN** 系统最终只应用一次业务影响，不产生重复计数

#### Scenario: Notification failure does not roll back a like
- **WHEN** 点赞记录已经成功提交但通知事件处理失败
- **THEN** 点赞结果保持成功，通知处理通过重试或死信流程恢复
