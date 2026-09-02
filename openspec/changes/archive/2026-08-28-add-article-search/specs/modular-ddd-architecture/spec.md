## MODIFIED Requirements

### Requirement: 系统按五个限界上下文组织业务能力
系统 SHALL 将业务能力组织为 User、Article、Comment、Like、Notification 和 Search 六个限界上下文，并 SHALL 为每个上下文明确其写模型、派生读模型和业务规则的所有权。Search SHALL 只拥有可重建的搜索投影，不得拥有或修改 Article 的文章业务状态。

#### Scenario: 业务代码完成上下文归属
- **WHEN** 检查业务包结构
- **THEN** 用户、文章、评论、点赞、通知和搜索业务代码分别归属于对应的六个上下文
- **AND** 不存在继续承载多个上下文业务规则的全局 Service、Repository 或 Model 包

#### Scenario: 上下文拥有自己的写模型
- **WHEN** 一个用例需要修改用户、文章、评论、点赞关系或通知数据
- **THEN** 对应数据只能由拥有该写模型的上下文执行修改
- **AND** 其他上下文不得直接写入该上下文拥有的表、集合或缓存关系

#### Scenario: Search 只维护派生搜索投影
- **WHEN** Search 查询、增量同步或全量重建处理文章数据
- **THEN** Search 只读取 Article 拥有的文章数据或其数据库变更并维护 Elasticsearch 投影
- **AND** Search 不通过 Repository、SQL 或其他方式修改 Article 拥有的文章表和业务状态

#### Scenario: Article 写入不依赖 Search
- **WHEN** Article 创建、编辑、发布、删除或恢复文章
- **THEN** Article 只提交自身业务数据
- **AND** 不同步调用 Search 或 Elasticsearch 完成文章写入

### Requirement: 架构和兼容要求必须可验证
系统 SHALL 提供自动化检查或测试，验证层级依赖、上下文隔离、事务原子性、搜索投影边界和外部契约兼容性。

#### Scenario: 上下文迁移完成
- **WHEN** 任一上下文切换到新模块并准备删除旧实现
- **THEN** 该上下文的关键行为特征测试、Domain/Application 测试、Adapter 测试和架构检查均通过
- **AND** 旧测试仅在已有等价覆盖时删除

#### Scenario: 全部重构完成
- **WHEN** 六个上下文和 Platform 全部完成组装
- **THEN** 全量测试、架构依赖检查、HTTP/gRPC/Kafka 契约对比、搜索契约测试和跨上下文事务回滚测试均通过
- **AND** 旧全局技术分层业务目录和旧内部 `pkg` 技术路径不再被业务代码依赖

#### Scenario: Search 依赖边界检查
- **WHEN** 检查 Search 上下文和平台层 Canal Client 的依赖
- **THEN** Search Application 通过 Port 使用 Elasticsearch、文章索引数据源和 Canal 事件输入
- **AND** 平台层 Canal Client 不依赖 Search、Article 或其他业务上下文

## RENAMED Requirements

- FROM: `系统按五个限界上下文组织业务能力`
- TO: `系统按六个限界上下文组织业务能力`
