## Why

当前前台只能通过文章列表浏览内容，无法按标题、正文或标签快速定位已发表文章。需要引入独立的 Search 上下文，以 Elasticsearch 提供中文和标题拼音检索，并通过 Canal 订阅 MySQL binlog，在不耦合 Article 写入流程的前提下维护可重建的搜索索引。

## What Changes

- 新增前台文章搜索接口，仅检索已发表文章，支持标题原文、标题完整拼音、标题拼音首字母简写、正文纯文本和标签查询，并固定按相关性排序。
- 搜索结果返回原始标题、原文命中时的标题高亮、正文命中时的纯文本高亮摘要、标签及分页总数；拼音命中不强制映射为中文标题高亮。
- 新增独立的 Search 限界上下文，负责搜索查询、索引文档转换、增量同步和全量重建，不拥有或修改 Article 的业务状态。
- 在平台层新增通用 Canal Client，统一负责连接、批次拉取、确认、回滚、重连和生命周期管理；Search 上下文只处理文章变更语义。
- 新增 `search-sync` 长期运行入口，直接消费 Canal 文章变更并写入 Elasticsearch；仅在标题、正文、标签或状态变化时处理索引，忽略单纯的浏览量、点赞数和评论数更新。
- 新增 `search-rebuild` 一次性运行入口，批量读取 MySQL 中已有的已发表文章并重建 Elasticsearch 索引。
- 正文索引前解析 Markdown，仅保留普通文本，排除 Markdown 标记、图片及地址、代码块和行内代码。
- 标签在索引前去除首尾空格、排除空值、按英文大小写不敏感规则去重，并支持常规文本分词；本次不提供标签同义词。
- 接受 MySQL 提交到 Elasticsearch 可见之间数秒级的最终一致性延迟；文章写入不依赖 Elasticsearch，搜索不可用时不降级为 MySQL 模糊查询。

## Capabilities

### New Capabilities

- `article-search`: 定义已发表文章的搜索范围、查询语义、拼音能力、高亮响应、索引同步、全量重建和最终一致性要求。

### Modified Capabilities

- `modular-ddd-architecture`: 在现有模块化单体中增加独立的 Search 限界上下文，并明确其与 Article、平台层 Canal Client 及 Elasticsearch 的依赖边界。

## Impact

- 本 change 的全部代码、配置、测试和新增文档 MUST 遵守仓库根目录 `AGENTS.md`；实施前需重新读取该文件，并以其中的分层、命名、中文注释、错误处理、资源生命周期、测试和交付规范作为强制约束。
- 新增 `internal/search` 上下文及其 Domain、Application、Infrastructure、Interfaces 和模块组装代码。
- 新增平台层 Canal Client；新增 Elasticsearch 客户端、Markdown 纯文本提取及文章索引数据源适配器。
- 新增公开 HTTP 接口 `GET /article/search`，并在现有 `server` 入口组装 Search 查询能力。
- 新增 `search-sync` 和 `search-rebuild` Cobra 运行入口，分别承担 Canal 增量同步和 MySQL 全量重建。
- 扩展应用配置、Docker Compose 和生产部署配置，以提供 Elasticsearch、Canal 连接及独立同步进程所需参数。
- 引入 Canal Go Client、Elasticsearch Client 和 Markdown Parser 相关依赖，并增加搜索查询、文档转换、事件处理、重建流程和接口契约测试。
- 不修改文章公开写入 API、数据库结构、文章状态语义及现有 Kafka 消息格式。
