## Context

当前系统是同一仓库、同一二进制的模块化单体，已经通过 `server`、`grpc` 和 `kafka-consume` 等 Cobra 子命令区分运行入口。Article 上下文以 MySQL `articles` 表作为文章数据真相，标题、Markdown 正文、逗号分隔标签、文章状态和互动统计均存储在同一行；实际领域状态为 `1-已删除、2-草稿、3-已发表`。

开发环境已经提供 Elasticsearch 8.19.20、IK 与拼音插件、Canal Server 1.1.8，以及满足 Canal 要求的 MySQL ROW/FULL binlog 配置，但应用尚未包含 Elasticsearch 配置、Canal Client、搜索模块或索引初始化能力。Canal 当前只监听 `blog.articles`，而浏览、点赞和评论都会频繁更新该表中的统计字段，因此同步处理必须主动跳过与搜索无关的更新。

参见 `proposal.md` 了解变更动机，参见 `specs/article-search/spec.md` 和 `specs/modular-ddd-architecture/spec.md` 了解行为契约。

## Goals / Non-Goals

**Goals:**

- 使本 change 的实现全过程遵守仓库根目录 `AGENTS.md`，而不是仅在交付前补做格式或注释检查。
- 在现有模块化单体内建立职责单一的 Search 上下文，并保持 Article 对 Search 和 Elasticsearch 无感知。
- 让在线查询、Canal 增量同步和 MySQL 全量重建复用同一份搜索文档转换规则。
- 通过平台层 Canal Client 统一封装协议连接、批次确认、回滚、重连和退出行为，业务上下文只处理事件语义。
- 使用版本化物理索引和稳定别名，使首次导入、重复重建和回滚不要求修改搜索调用方。
- 使 `server`、`search-sync` 和 `search-rebuild` 各自只初始化运行所需的资源。

**Non-Goals:**

- 不拆分独立仓库、独立 HTTP 微服务或 Search 与 Article 之间的 RPC。
- 不引入 Kafka、Outbox、双写或分布式事务来维护搜索索引。
- 不支持管理端搜索、拼写纠错、搜索联想、标签同义词、任意子串标签匹配、热度排序或最新排序。
- 不使用 Elasticsearch 作为文章详情或业务状态的数据真相，也不从 Search 修改 MySQL 文章数据。
- 不提供 MySQL `LIKE` 搜索降级，不在本次变更中建设通用多实体搜索框架。

## Decisions

### 1. Search 作为独立上下文留在模块化单体中

新增 `internal/search`，按现有项目风格组织 Domain、Application、Infrastructure、Interfaces 和 `module.go`。Search 负责文章搜索文档、查询条件、索引同步决策、全量重建编排和 HTTP 适配；Article 继续独占文章生命周期及 MySQL 写入。

Search 不复用 Article Repository，也不调用 Article Application Service。全量重建所需文章数据由 Search 声明最小化的只读 `ArticleSource` Port，Infrastructure 使用调用方拥有的查询模型读取 `articles` 表。这样可以避免 Search 反向依赖 Article 的持久化模型，同时符合现有跨上下文只读查询约束。

备选方案是把搜索放进 `internal/article`。该方案文件更少，但会让 Article 同时承担 Elasticsearch mapping、Canal 事件、Markdown 清理和索引重建，扩大上下文职责，因此不采用。

### 2. 通用 Canal Client 放在平台层

新增 `internal/platform/canal`，只提供与业务无关的能力：

- 建立和关闭 Canal 连接；
- 订阅指定 destination；
- 顺序拉取 batch；
- 在业务处理成功后 ack；
- 在业务处理失败后 rollback；
- 连接中断时按有上限的退避策略重连；
- 响应 `context.Context` 取消和进程退出信号；
- 将 Canal 原始批次交给注册的批次处理器。

平台层不得解析“已发表文章”等业务含义，也不得依赖 Search 或 Article。Search 的增量同步 Adapter 负责过滤 schema/table、识别 INSERT/UPDATE/DELETE、解析文章字段和调用应用用例。

备选方案是把 Client 全部放入 Search Infrastructure。它对当前功能可行，但连接、ack 和重连属于稳定的通用技术能力，放入平台层更符合仓库现有 Kafka、数据库和 OSS 客户端的组织方式。

### 3. 使用三个运行入口隔离资源和生命周期

沿用现有 Cobra 模式增加：

| 入口 | 初始化资源 | 职责 |
| --- | --- | --- |
| `server` | Elasticsearch 查询 Client、现有 HTTP 依赖 | 注册 `GET /article/search` 并执行在线搜索 |
| `search-sync` | Canal Client、Elasticsearch 写入 Client | 长期顺序消费 Canal 批次并维护索引 |
| `search-rebuild` | MySQL、Elasticsearch 写入 Client | 一次性分批构建新物理索引并切换别名 |

`search-sync` 以单实例运行，同一个 destination 不进行多实例并发消费。`server` 创建 Elasticsearch Client 时不强制通过启动探测阻断整个 HTTP 服务；Elasticsearch 不可用只使搜索接口返回服务错误。`search-sync` 和 `search-rebuild` 则在必要依赖不可用时启动失败或返回非零退出码。

备选方案是在 `server` 内启动 Canal goroutine。该方案会把 HTTP 生命周期和同步生命周期绑定，并在 HTTP 多副本部署时引入重复消费者，因此不采用。

### 4. Search Application 通过最小 Port 编排三类用例

Search Application 声明以下最小能力：

- `IndexSearcher`：执行文章搜索并返回命中、高亮和总数；
- `IndexWriter`：创建映射、批量 upsert、按文章 ID 删除、刷新和切换别名；
- `ArticleSource`：按稳定游标分批读取已发表文章；
- `TextExtractor`：把 Markdown 转换为可搜索纯文本；
- `ArticleChangeHandler`：接收已解析的文章行变更并作出 upsert、delete 或 ignore 决策。

在线查询、增量同步和全量重建共用一个文档工厂，确保标题、正文和标签转换结果一致。Canal 原始协议对象停留在 Interfaces/Infrastructure 边界，不进入 Domain 或 Application DTO。

### 5. 搜索文档只保存查询和展示所需字段

搜索文档采用文章 ID 作为 Elasticsearch `_id`，建议字段如下：

| 字段 | 类型/用途 |
| --- | --- |
| `article_id` | `unsigned_long` 或可无损表示 `uint64` 的 keyword；用于响应和校验 |
| `title` | 原始标题；中文/英文原文检索和高亮 |
| `title.pinyin` | `title` 的 multi-field；完整拼音检索，不用于高亮 |
| `title.pinyin_initial` | `title` 的 multi-field；连续拼音首字母简写检索，不用于高亮 |
| `content` | 清理后的正文纯文本；检索和高亮摘要 |
| `tags` | 规范化后的标签数组；常规文本分词和响应展示 |
| `updated_time` | 日期；用于诊断文档新旧，不参与本次排序 |

不保存 Markdown 原文、文章状态、互动统计、图片地址或代码内容。状态只用于决定文档是否存在，文章详情仍从 Article 接口读取。

物理索引使用版本化名称，例如 `article_search_20260827093000`；应用统一访问别名 `article_search`。Mapping 使用显式字段定义并关闭无意的动态字段扩张，避免 Canal 字段变化自动污染索引。

备选方案是使用固定单索引并在重建时清空。该方案更简单，但重建失败会破坏当前可用索引且产生搜索中断，因此采用版本化索引与别名。

### 6. 标题使用原文、完整拼音和首字母简写子字段

`title` 使用 IK 中文分析器完成原文检索；`title.pinyin` 使用完整拼音分析器，同时保留逐词完整拼音和连续完整拼音，并关闭首字母输出；`title.pinyin_initial` 使用独立首字母分析器，开启 `keep_first_letter`、关闭 `keep_separate_first_letter`，只生成 `srlj` 形式的连续首字母，并限制首字母最大长度。

查询时分别对原文、完整拼音和首字母字段构造 match 子句，完整拼音多词查询要求全部查询词满足，保证 `shen ru li jie`、`shenrulijie` 和 `srlj` 的基础能力。首字母字段独立设置较低权重，避免短简写产生过多错误召回。

高亮只配置原始 `title` 和 `content`，不使用 `matched_fields` 把拼音 offset 强行映射到中文标题。若只有 `title.pinyin` 或 `title.pinyin_initial` 命中，则返回原始标题并让 `title_highlight` 为空。

备选方案是把首字母 token 混入现有 `title.pinyin`。该方案字段更少，但无法独立控制首字母召回和权重，因此使用单独的 `title.pinyin_initial` multi-field。前缀和纠错字段仍不属于本次范围。

### 7. Markdown 通过 AST 提取纯文本

引入 Markdown Parser，将正文解析为 AST 后按节点类型遍历：

- 保留文档标题、段落、引用、列表及强调节点中的普通文本；
- 保留链接显示文字但丢弃 URL；
- 跳过图片节点、围栏代码块、缩进代码块和行内代码；
- 在块级节点之间插入空格或换行，避免相邻文字错误粘连；
- 对最终文本做空白归一化。

不使用正则表达式批量删除 Markdown，因为嵌套标记、转义、链接和代码围栏容易产生错误残留。文档转换失败时，增量同步不得 ack 当前批次；全量重建则停止并返回包含文章 ID 的错误。

### 8. 标签规范化与展示使用同一数组

文档工厂按输入顺序处理逗号分隔标签：先 `TrimSpace`，排除空标签，再使用小写值作为去重 key，保留第一次出现的规范化展示值。例如 `Go, go ,Web` 得到 `Go,Web`。Elasticsearch 分析阶段再统一小写，使不同英文大小写的查询结果一致。

标签使用支持中英文常规分词的文本字段参与查询，不建立 n-gram 任意子串字段，也不配置 `Go`/`Golang` 同义词。后续如需同义词，应仅扩展标签查询，不改变 MySQL 原始标签。

### 9. 固定查询结构和响应转换

Search Application 将已校验关键词构造成多个 should 子句，至少一个子句命中：

- 标题原文：boost `5`；
- 标题完整拼音：boost `3`；
- 标签：boost `3`；
- 标题拼音首字母：boost `2`；
- 正文纯文本：boost `1`。

本次使用 `from/size` 支持页码分页，并启用精确命中总数。由于 `page_size` 最大为 20，博客数据规模下先不引入 PIT 或 `search_after`；超过 Elasticsearch 可分页窗口的深分页不作为本次保证能力。

高亮器只请求一个正文片段并限制片段长度，Search Application 将 Elasticsearch 高亮结果转换为稳定 DTO：`id`、`title`、`title_highlight`、`summary`、`tags`。若高亮不存在则返回空字符串，而不是回退为完整正文。

### 10. 根据事件类型和变更字段决定索引操作

Canal 使用 MySQL FULL row image，Search 事件 Adapter 将字段名映射为文章变更模型，并依据下表处理：

| 事件 | 条件 | 操作 |
| --- | --- | --- |
| INSERT | 新状态为 3 | upsert |
| INSERT | 新状态不是 3 | ignore |
| UPDATE | 状态改为非 3 | delete |
| UPDATE | 新状态为 3，且状态/title/content/tags 任一变化 | upsert |
| UPDATE | 仅统计或其他非搜索字段变化 | ignore |
| DELETE | 任意旧状态 | delete |

状态值通过语义清晰的来源状态常量表达，不散落魔法数字。upsert 使用文章 ID 作为 `_id`，delete 对不存在文档视为成功，因此重复投递是幂等的。

每个 batch 按原始顺序处理；只有所有必要操作成功后平台层才 ack。失败时 rollback 并按有上限的指数退避重试，进程不跳过无法处理的文章事件。这样优先保证不丢变更，代价是持续异常事件会阻塞后续同步，需要通过日志和监控人工处理。

### 11. 全量重建使用新索引和原子别名切换

`search-rebuild` 执行以下流程：

1. 校验别名、mapping 和所需分析插件；
2. 创建新的版本化物理索引；
3. 使用文章 ID 游标分批读取 `status=3` 的文章；
4. 使用与增量同步相同的文档工厂转换并 Bulk 写入；
5. 刷新新索引并执行基本数量校验；
6. 原子地把 `article_search` 别名从旧索引切换到新索引；
7. 保留上一个物理索引用于短期回滚，旧索引清理由显式运维动作完成。

为避免重建期间增量只写入旧索引，部署和人工重建时先停止 `search-sync`，再执行重建，别名切换成功后恢复 `search-sync`。停止期间 Canal 保留消费位点；重启后积压事件按顺序回放到新别名，修正全量扫描期间发生的新增、修改和删除。

若新索引构建失败，不切换别名并删除失败的新索引；若切换后发现异常，可停止同步并把别名切回上一个索引，再修复后重新重建。

### 12. 配置和部署区分开发与生产安全要求

应用配置新增 Elasticsearch 与 Canal 配置段，包含地址、索引别名、请求超时、destination、账号、密码、batch 大小、空批次等待和重连退避等参数。敏感字段只通过环境变量注入，示例配置使用占位值。

开发 Compose 延续单节点 ES、IK/拼音插件和关闭认证的本地配置，并增加 `search-sync` 服务。生产 Compose/Kubernetes 必须补齐 MySQL ROW/FULL binlog、Canal、Elasticsearch 持久化、搜索同步进程以及网络访问控制；不得把开发环境中无认证且公开暴露的 Elasticsearch 配置直接用于生产。

### 13. `AGENTS.md` 作为实施阶段强制规范

实施本 change 前必须重新读取仓库根目录 `AGENTS.md`，并在代码分析、设计落地、编码、测试和交付全过程遵守其约束。该约束至少包括：

- 只修改当前搜索功能必需的文件，不顺手重构、批量改名或格式化无关代码；
- Search、Article、Platform、Handler、Application、Domain 和 Infrastructure 依赖保持本设计及仓库现有分层，不把业务规则放入 HTTP、命令入口或平台客户端；
- 所有手写结构体字段添加准确的中文行尾注释，状态、时间、数值单位和枚举取值说明完整；
- 每个手写具名函数或方法添加功能注释，导出注释以标识符开头，函数体主要流程按 `1.`、`2.`、`3.` 连续编号且最多使用二级编号；
- 参数数量大于等于 5 个的函数补充逐项参数说明，参数过多且职责不清时优先改用请求或配置结构体；
- 错误不得静默吞掉，包装时保留 `%w` 错误链；Canal 连接、Elasticsearch 响应体、goroutine、锁和其他资源必须在成功及异常路径成对释放；
- 不直接修改自动生成文件，不提交敏感信息，示例配置只使用安全占位值；
- 所有修改的 Go 文件执行 `gofmt`，业务变化同步增加正常、失败和边界测试，最终执行 `go test ./...`；若环境限制导致全量测试无法完成，交付说明必须如实记录原因和风险。

实施过程中若本设计中的示例目录或技术细节与 `AGENTS.md` 冲突，以 `AGENTS.md` 为准，但不得借此改变已经由增量 spec 确定的外部行为。代码评审和最终验收应把这些规范作为完成条件，而不是可选建议。

## Risks / Trade-offs

- [最终一致性导致短暂旧结果] → 接受数秒延迟，保持 Article 写入与搜索故障隔离；详情接口继续以 MySQL 文章状态为准。
- [直接 Canal Client 为单消费者，无法水平扩展同步吞吐] → 当前文章规模使用单实例；通过 batch 和 Elasticsearch Bulk 提高吞吐，出现真实瓶颈后再评估 Kafka。
- [持续失败的文章事件会阻塞后续批次] → 记录 destination、batch ID、文章 ID 和根因，使用有上限退避避免忙循环，并提供全量重建作为修复手段；本次不引入 DLQ。
- [拼音首字母简写可能产生额外召回或排序噪声] → 使用独立 `title.pinyin_initial` 字段、限制首字母长度并将 boost 降为 `2`，保持原文标题和完整拼音优先。
- [Markdown AST 提取与页面渲染文字边界存在差异] → 为图片、链接、嵌套强调、列表、引用、代码块和行内代码建立固定测试样例。
- [重建时未暂停同步可能向旧索引写入并在切换后丢失增量] → 将“停止同步、重建、恢复同步”写入标准部署步骤，并依靠 Canal 位点回放积压事件。
- [MySQL 初始化脚本中的文章状态注释与实际领域状态不一致] → 实施时同步修正文档性注释和测试基准，不改变表结构或既有数据值。
- [生产 Elasticsearch 未启用访问控制会暴露全文数据和管理 API] → 生产配置启用认证、限制网络访问并通过 Secret 注入凭证。
- [实施过程中只关注功能而遗漏仓库代码规范] → 开工前、分阶段自检和最终验收均对照根目录 `AGENTS.md`，任务清单单独记录规范读取与合规检查。

## Migration Plan

1. 增加配置、平台层 Elasticsearch/Canal 能力、Search 上下文和三个运行入口，但暂不开放搜索路由。
2. 部署或升级 MySQL binlog、Canal、Elasticsearch 及所需 IK/拼音插件，验证版本兼容和持久化配置。
3. 保持 `search-sync` 停止，运行 `search-rebuild`，把现有已发表文章导入新物理索引并创建 `article_search` 别名。
4. 启动单实例 `search-sync`，确认 Canal 位点推进、统计字段更新被忽略、文章发布/编辑/删除能正确 upsert 或 delete。
5. 在 `server` 中开放 `GET /article/search`，执行中文、完整拼音、拼音首字母简写、正文、标签、高亮、分页和故障场景验收。
6. 观察同步延迟、错误率、ES 查询耗时和索引文档数；稳定后保留一个可回滚旧索引并显式清理更早索引。

回滚时先关闭搜索路由和 `search-sync`，保持 Article 写入不受影响；如需恢复旧搜索数据，将别名切回上一物理索引。移除搜索功能不需要回滚 MySQL 业务数据或数据库结构。
