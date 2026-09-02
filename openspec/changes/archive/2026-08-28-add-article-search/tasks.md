## 1. 实施规范、依赖与运行配置

- [x] 1.1 实施前重新阅读仓库根目录 `AGENTS.md`，把最小修改范围、分层依赖、目录命名、中文注释、错误与资源处理、测试和交付检查项作为本 change 的强制完成标准。
- [x] 1.2 选定并加入与当前 Go/Elasticsearch/Canal 版本兼容的 Elasticsearch Client、Canal Go Client 和 Markdown AST Parser 依赖，确认不引入重复或废弃 SDK。
- [x] 1.3 在平台配置中新增 Elasticsearch 与 Canal 配置结构，覆盖地址、认证、索引别名、超时、destination、batch 大小、空批次等待和重连退避，并为每个手写结构体字段补充中文行尾注释。
- [x] 1.4 更新本地和容器 YAML、`.env.example` 的安全占位配置，并增加环境变量展开、缺失必填项和默认值测试。
- [x] 1.5 修正文章建表初始化脚本中状态注释与实际 `1-已删除、2-草稿、3-已发表` 语义不一致的问题，不修改表结构和数据值。

## 2. 平台层 Elasticsearch 与 Canal 客户端

- [x] 2.1 新增平台层 Elasticsearch Client 构造与关闭能力，支持配置地址、认证和请求超时，并允许 `server` 在不执行强制启动探测的情况下完成组装。
- [x] 2.2 新增平台层 Canal Client，封装连接、订阅、顺序拉取 batch、ack、rollback、重连、退避和 `context.Context` 取消退出。
- [x] 2.3 为 Canal Client 定义与业务无关的批次处理器接口，确保只有业务处理成功才 ack，失败时 rollback 且不推进消费位点。
- [x] 2.4 为 Canal Client 增加成功确认、失败回滚、空批次等待、连接中断重连和取消退出测试，并验证平台层不依赖任何业务上下文。

## 3. Search 上下文基础模型与文档转换

- [x] 3.1 创建 `internal/search` 上下文及模块骨架，按实际需要组织 Domain、Application、Infrastructure 和 Interfaces，并在架构检查中登记第六个上下文。
- [x] 3.2 定义搜索文档、搜索条件、搜索命中、分页响应和来源文章变更模型，以及 `IndexSearcher`、`IndexWriter`、`ArticleSource`、`TextExtractor` 等最小 Port。
- [x] 3.3 实现 Markdown AST 纯文本提取 Adapter，保留标题、段落、列表、引用、强调和链接显示文字，排除 Markdown 标记、URL、图片、围栏/缩进代码块和行内代码，并归一化空白。
- [x] 3.4 实现标签规范化：按英文逗号拆分、去除首尾空格、排除空标签、使用小写 key 去重并保留首次出现的展示值。
- [x] 3.5 实现统一搜索文档工厂，让增量同步和全量重建复用标题、正文、标签和更新时间转换规则，并在转换错误中保留文章 ID 和原始错误链。
- [x] 3.6 为 Markdown 提取、标签规范化和文档工厂增加表驱动测试，覆盖嵌套标记、链接、图片、代码、空标签、大小写重复和空正文边界。

## 4. Elasticsearch 索引和查询 Adapter

- [x] 4.1 定义版本化文章搜索索引 mapping：文章 ID、原始标题、完整拼音 multi-field、纯文本正文、规范化标签和更新时间，并禁止无意的动态字段扩张。
- [x] 4.2 配置标题原文 IK 分析器和完整拼音分析器，支持带空格及连续完整拼音，关闭首字母能力，并为标签配置大小写不敏感的常规中英文分词。
- [x] 4.3 实现索引写入 Adapter，支持创建物理索引、Bulk upsert、幂等 delete、refresh、数量检查和原子别名切换，并在 Bulk 部分失败时返回可定位文档的错误。
- [x] 4.4 实现搜索查询 Adapter，按标题原文 `5`、标题拼音 `3`、标签 `3`、正文 `1` 的权重执行固定相关性搜索，使用 `from/size` 分页并返回精确命中总数。
- [x] 4.5 仅对原始标题和正文请求高亮，将拼音单独命中映射为空标题高亮，将正文命中映射为单个有限长度的纯文本摘要。
- [x] 4.6 使用可控 HTTP 测试服务或 Elasticsearch 集成环境验证 mapping、查询 JSON、Bulk 部分失败、幂等删除、别名切换和搜索错误映射。
- [x] 4.7 增加 Elasticsearch analyzer 集成测试，验证中文标题、带空格完整拼音、连续完整拼音、英文标签大小写和标签常规分词的实际命中结果。

## 5. Canal 文章增量同步

- [x] 5.1 实现 Search 的 Canal 事件 Adapter，只接收目标 schema/table 的文章 INSERT、UPDATE、DELETE，并把 FULL row image 转换为来源文章变更模型。
- [x] 5.2 根据实际文章状态常量实现同步决策：已发表 INSERT/upsert、草稿发布/upsert、退出发表状态/delete、物理删除/delete、已发表搜索字段变更/upsert。
- [x] 5.3 使用 Canal 字段更新标记识别 `title`、`content`、`tags` 和 `status` 变化；仅统计字段或其他非搜索字段变化时直接 ignore，禁止访问 Elasticsearch。
- [x] 5.4 保证同一 batch 按原始顺序处理并复用统一文档工厂；必要操作全部成功后返回成功，任一转换或 Elasticsearch 操作失败时返回错误以触发 rollback。
- [x] 5.5 为 INSERT、草稿发布、已发表编辑、退出发表状态、物理删除、统计字段更新、重复事件、非目标表和畸形行数据增加同步回归测试。

## 6. 全量重建能力

- [x] 6.1 实现 Search 自有的 MySQL `ArticleSource` Adapter，按文章 ID 游标分批读取 `status=3` 的最小字段集合，不复用 Article Repository 或持久化模型。
- [x] 6.2 实现全量重建 Application 用例：创建新版本索引、分页转换、Bulk 写入、刷新和基本数量校验、原子切换 `article_search` 别名。
- [x] 6.3 在重建失败时保持现有别名不变并清理失败的新索引；切换成功后保留上一个索引用于显式回滚，不自动删除所有旧索引。
- [x] 6.4 为无旧索引首次构建、已有别名重建、空文章库、分页边界、文档转换失败、Bulk 失败和别名切换失败增加测试。

## 7. HTTP 接口、模块组装与运行入口

- [x] 7.1 实现 Search Application 查询用例，校验并裁剪 `keyword`，校验 `page>=1` 和 `10<=page_size<=20`，并把 Infrastructure 命中转换为稳定响应 DTO。
- [x] 7.2 实现公开 `GET /article/search` HTTP Adapter 和路由，返回 `id`、`title`、`title_highlight`、`summary`、`tags`、`total`、`page` 和 `page_size`，搜索失败时不执行 MySQL 降级。
- [x] 7.3 实现 Search `module.go`，在 `server` 组合根中只组装在线查询依赖，并更新 AppHandler/路由组而不改变现有文章接口行为。
- [x] 7.4 新增 `search-sync` Cobra 子命令，组装平台 Canal Client、Search Canal Adapter 和 Elasticsearch 写入 Adapter，处理系统信号并保证资源成对关闭。
- [x] 7.5 新增 `search-rebuild` Cobra 子命令，组装 MySQL、文档转换和 Elasticsearch 重建依赖，完成后返回明确退出状态并释放资源。
- [x] 7.6 为 HTTP 参数边界、原文标题高亮、拼音命中无标题高亮、正文摘要、仅标签命中空摘要、精确总数和 Elasticsearch 不可用增加 Handler/Application 测试。

## 8. 部署、运维与文档

- [x] 8.1 更新开发 Docker Compose，确保 Elasticsearch、IK/拼音插件、Canal destination 和 `search-sync` 服务配置一致，并让同步进程等待必要依赖就绪。
- [x] 8.2 更新生产 Compose 和 Kubernetes 清单，补齐 MySQL ROW/FULL binlog、Canal、Elasticsearch 持久化、单副本 `search-sync`、Secret 注入和网络访问控制。
- [x] 8.3 编写首次上线和日常重建说明，明确“停止 search-sync → 运行 search-rebuild → 验证并切换别名 → 恢复 search-sync → 检查积压回放”的顺序。
- [x] 8.4 编写索引回滚和故障排查说明，覆盖别名切回、失败新索引清理、Canal batch 阻塞、同步延迟、插件缺失和 ES 无认证风险。
- [x] 8.5 更新架构文档，说明 Search 是模块化单体中的第六个上下文、Article 是数据真相、Elasticsearch 是可重建投影、平台层 Canal Client 不包含业务规则。

## 9. 验证与验收

- [x] 9.1 对照仓库根目录 `AGENTS.md` 逐项检查本次变更：修改范围、分层依赖、目录和包命名、全部手写结构体字段注释、具名函数注释、编号流程注释、五个及以上参数说明、错误链、资源释放及敏感信息均符合规范；随后对全部修改的 Go 文件执行 `gofmt`。
- [x] 9.2 执行 Search、Platform Canal、配置、HTTP 路由和命令相关包测试，并执行 `go test ./...` 与 `scripts/check_architecture.sh`。
- [x] 9.3 在容器集成环境完成首次全量重建，并验证中文标题、两种完整拼音、正文纯文本、标签大小写、相关性顺序、高亮和分页契约。
- [x] 9.4 在容器集成环境验证文章发布、标题/正文/标签修改、转草稿、软删除、物理删除和统计字段自增对应的 ES upsert/delete/ignore 行为及数秒级同步延迟。
- [x] 9.5 验证 Elasticsearch 故障不影响 Article 创建、编辑和发布，搜索接口返回服务错误；恢复后 Canal 未确认批次能够继续处理。
- [x] 9.6 检查最终 diff，确认没有敏感信息、无关重构、生成文件手工修改或现有公开 API/Kafka/数据库结构兼容性变化，并执行 `openspec validate add-article-search --strict`。

## 10. 标题拼音首字母简写

- [x] 10.1 在文章搜索索引中新增独立 `title.pinyin_initial` multi-field 和首字母 analyzer，生成 `srlj` 形式的连续首字母并限制最大长度，不改变现有完整拼音字段语义。
- [x] 10.2 在固定相关性查询中增加 `title.pinyin_initial` 匹配子句，boost 设置为 `2`，仅首字母命中时保持原始标题且不生成中文标题高亮。
- [x] 10.3 更新 Elasticsearch 请求契约和 Search Application 测试，覆盖 `srlj` 命中“深入理解”、首字母字段权重以及空高亮行为。
- [x] 10.4 执行 `search-rebuild` 创建新物理索引并切换别名，在真实 Elasticsearch 中验证完整拼音能力保持兼容且首字母简写可命中。
