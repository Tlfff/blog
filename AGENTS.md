# AGENTS.md

## 交流约定

- 所有会话回复、以及新增的说明性文档，一律使用**中文**。
- 代码注释沿用仓库既有风格（见 `docs/note/comment-style-guide.md`），保持中文注释。

## 项目结构约定

- 严格四层架构：`Interfaces → Application → Domain`，`Infrastructure` 实现 `Domain` 定义的 Port。
- 三个界限上下文：`identity`、`content`、`community`（通知为 Community 内部子模块）。
- 领域层不得导入 GORM/Redis/MongoDB/Kafka/MinIO/Gin 等框架。
- 详细约定见 `DDD实践.md` 与 `docs/architecture/layering-rules.md`。
