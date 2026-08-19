## MODIFIED Requirements

### Requirement: Notification creation from business events

Notification 上下文 SHALL 根据约定的点赞、评论、回复等业务事件创建通知，并 SHALL 支持事件重复投递而不生成重复通知。Notification 上下文不得要求业务操作与通知写入处于同一事务。

#### Scenario: Article like creates an author notification
- **WHEN** 用户点赞他人文章且事件包含合法发送方、接收方和文章信息
- **THEN** Notification 上下文为文章作者创建一条对应的未读通知

#### Scenario: Self-like does not notify the author
- **WHEN** 用户点赞自己的文章
- **THEN** 系统不创建通知

#### Scenario: Duplicate notification event is ignored
- **WHEN** 同一个业务事件被重复消费
- **THEN** 系统最多保留一条对应通知，并保持未读数量正确

### Requirement: Notification query and read state

Notification 上下文 SHALL 支持当前用户未读数量查询、分页通知列表和一键清空未读，并 SHALL 确保用户只能访问和修改自己的通知。

#### Scenario: User lists own notifications
- **WHEN** 已登录用户请求通知列表
- **THEN** 系统按约定分页返回该用户的通知，不返回其他用户的数据

#### Scenario: User clears unread notifications
- **WHEN** 已登录用户执行清空未读
- **THEN** 该用户现有未读通知被标记为已读，未读数量返回为零

#### Scenario: Unauthenticated notification access is rejected
- **WHEN** 未登录调用方请求通知数量、通知列表或清空未读
- **THEN** 系统拒绝请求且不泄露通知数据

### Requirement: Notification processing is isolated from main actions

通知事件处理失败 SHALL 支持重试和死信处理；通知写入或消费异常不得回滚已经成功完成的文章、评论或点赞主流程。Notification 上下文 SHALL 通过版本化事件契约接收其他上下文的业务结果。

#### Scenario: Notification consumer retries a transient failure
- **WHEN** 通知事件因临时存储错误处理失败
- **THEN** 消费者按既定策略重试，并在超过重试上限后转入死信处理

#### Scenario: Main action succeeds while notification is delayed
- **WHEN** 点赞或评论主流程已成功，但 Notification 上下文暂时不可用
- **THEN** 主流程保持成功，通知可在服务恢复后异步补处理
