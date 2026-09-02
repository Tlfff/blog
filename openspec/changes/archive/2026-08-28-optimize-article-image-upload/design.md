## Context

Article 当前通过初始化空草稿取得文章 ID，再批量生成 `article/<article_id>/...` 下的预签名 URL。正文保存完整公开 URL，硬删除依赖文章目录前缀清理对象。系统是单管理员、唯一文章作者模型，不需要额外的图片上传者授权。

项目已经具备 MinIO 预签名上传、公开域名配置、Goldmark Markdown 解析和本地事务协调能力。本设计在 Article 上下文内增加带可空文章归属的图片记录，不引入通用附件中心或关联表。

## Goals / Non-Goals

**Goals:**

- 允许文章创建前按单张图片实时上传。
- 使用稳定图片 ID 解耦正文和公开域名。
- 在创建和更新文章时可靠同步图片归属。
- 让公开详情和编辑详情只返回当前文章已绑定图片的映射。
- 在文章硬删除时精确清理已绑定图片。
- 兼容历史正文中的完整 URL 和外部图片 URL。

**Non-Goals:**

- 不实现 `article_id IS NULL` 图片的定时清理。
- 不实现上传完成确认、图片配额、媒体库或跨文章图片复用。
- 不迁移历史正文和历史 `article/<article_id>/...` 对象。

## Decisions

### 1. 使用带可空文章归属的 `article_images` 表

```sql
CREATE TABLE article_images (
    id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '图片ID',
    article_id   BIGINT UNSIGNED NULL COMMENT '所属文章ID，未绑定时为空',
    object_key   VARCHAR(255) NOT NULL COMMENT '对象存储Key',
    created_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (id),
    UNIQUE KEY uk_object_key (object_key),
    KEY idx_article_id (article_id)
);
```

`object_key` 唯一索引保证一个对象只对应一条图片记录；`article_id` 普通索引用于详情过滤和硬删除查询。沿用仓库现有建表风格，不增加数据库外键，关系完整性由 Application 事务和 Repository 条件更新保证。

图片只允许属于零篇或一篇文章。备选的多对多关联表支持复用，但增加关系同步和清理复杂度，当前单作者文章场景不需要。

### 2. 单图片上传先创建未绑定资源

使用 `POST /article/image/upload-url` 替代批量接口：

```json
// 请求
{"file_ext":"png"}

// 业务响应
{
  "image_id": 123,
  "upload_url": "https://minio.example/...",
  "url": "https://cdn.example/article/img/2026/08/<uuid>.png"
}
```

Application 校验扩展名，生成 `article/img/<year>/<month>/<uuid>.<ext>` 对象 Key并获取预签名 URL，随后创建 `article_id = NULL` 的图片记录。预签名生成失败时不写表；数据库写入失败时不向调用方返回上传凭证。

前端在 PUT 成功后把 `image://<image_id>` 写入原始 Markdown，并使用响应中的 `url` 实时预览。上传后未保存到文章的图片保持未绑定，等待后续独立清理能力处理。

### 3. 正文是图片引用集合，图片表保存当前归属

`articles.content` 原样保存 `image://<image_id>`，不保存当前公开域名。创建和更新请求不增加 `image_ids` 字段，Application 使用 Goldmark 从 Markdown 图片节点中提取格式严格为 `image://<正整数>` 的图片 ID。

正文表达展示位置，`article_images.article_id` 表达生命周期归属。保存时所有系统图片引用都必须存在，并且只能处于未绑定或已绑定当前文章状态；绑定其他文章的图片 ID 视为非法输入。

历史完整 URL 和外部图片 URL 不参与关系同步，保持原样。

### 4. 创建和更新文章事务内同步图片关系

Article Application 注入现有本地事务协调器；Article Repository 和图片 Repository 都从 `context.Context` 获取事务连接。

创建流程：

1. 校验文章字段并提取正文图片 ID。
2. 开启事务并创建文章，取得文章 ID。
3. 批量锁定图片记录，校验记录完整且 `article_id IS NULL`。
4. 将全部图片绑定到新文章并提交事务。

更新流程：

1. 读取文章原正文，分别提取旧图片 ID 和新图片 ID。
2. 计算新增、保留和移除集合。
3. 在事务中更新文章正文；将新增图片从未绑定状态绑定到当前文章，将移除图片从当前文章解绑为 `NULL`。
4. 任一图片不存在、已绑定其他文章或关系更新数量不符时回滚全部变更。

不立即删除解绑图片，避免编辑误操作造成不可恢复；后续定时清理只需处理超过宽限期且 `article_id IS NULL` 的记录。

### 5. 详情只映射正文引用且属于当前文章的图片

公开详情 `GET /article/detail` 与编辑详情 `GET /article/me/detail` 保持原始 `content`，并增加：

```json
"images": [
  {"id": 123, "url": "https://cdn.example/article/img/2026/08/<uuid>.png"}
]
```

详情查询对正文图片 ID 去重后，通过 `id IN (...) AND article_id = 当前文章ID` 一次批量查询。查询到的 `object_key` 使用当前公开域名配置生成 URL。

不存在、未绑定或绑定其他文章的图片 ID 不加入映射，但不阻断历史异常文章详情返回。备选方案是直接替换 `content`，但会导致编辑接口丢失稳定占位符，因此不采用。

### 6. 软删除保留关系，硬删除精确清理

软删除只改变文章状态，图片 `article_id` 保持不变，以支持文章恢复。

硬删除流程：

1. 校验文章已进入垃圾箱。
2. 按 `article_id` 查询全部已绑定图片。
3. 逐个删除 OSS 对象；任一失败时保留文章和图片记录并返回错误。
4. 对象全部清理成功后，在同一数据库事务中删除图片记录和文章记录。

OSS 与 MySQL 无法形成原子事务。如果对象删除成功后数据库事务失败，重试时对已不存在对象执行幂等删除，再继续清理数据库记录。

## Risks / Trade-offs

- [上传凭证已签发但 PUT 或文章保存失败会留下未绑定图片] → 本次保留记录，后续按 `article_id IS NULL` 和宽限期定时清理。
- [一张图片不能被多篇文章复用] → 通过单值 `article_id` 保持模型简单；需要复用时再升级为关联表。
- [文章更新同时修改正文和图片关系增加事务范围] → 批量查询和条件更新，并复用现有事务上下文，避免逐图片写入。
- [OSS 删除和数据库删除无法原子提交] → 先删对象、失败时保留数据库记录，并要求对象删除支持幂等重试。
- [历史正文没有图片记录] → 完整 URL 保持原样，不参与绑定、详情映射和新清理流程。

## Migration Plan

1. 创建包含可空 `article_id` 的 `article_images` 表，不修改 `articles` 表和历史数据。
2. 前端先支持 `content + images` 渲染模型，并继续兼容正文中的完整 URL。
3. 部署后端的新单图片接口、图片关系同步、详情映射和按关系硬删除能力。
4. 前端切换到 `POST /article/image/upload-url`，停止调用初始化和批量上传接口。
5. 确认调用方完成切换后移除旧路由和旧实现。

回滚到旧后端前若尚未保存占位符正文，可直接回滚；一旦已产生 `image://<image_id>` 正文，回滚版本必须保留详情图片映射能力，否则新正文中的图片无法展示。
