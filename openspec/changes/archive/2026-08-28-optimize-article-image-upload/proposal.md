## Why

当前文章图片上传必须先初始化空草稿并依赖文章 ID 生成对象目录，导致图片上传与文章生命周期耦合。将图片先作为未绑定资源实时上传，再在文章保存时建立归属，可以取消空草稿流程，并通过稳定图片占位符避免公开域名变化时批量修改正文。

## What Changes

- 新增文章图片记录，保存图片 ID、可空文章 ID、对象存储 Key 和创建时间。
- 将图片对象 Key 调整为与文章 ID 无关的 `article/img/<year>/<month>/<uuid>.<ext>`。
- 将批量上传凭证接口改为单图片实时上传凭证接口；申请凭证时创建 `article_id = NULL` 的图片记录，并返回图片 ID、预签名 URL 和当前公开 URL。
- 文章正文使用 `image://<image_id>` 保存图片引用，不持久化完整公开 URL。
- 创建文章时绑定正文引用的图片；更新文章时绑定新增图片，并将正文移除图片的 `article_id` 设回 `NULL`。文章保存与图片关系更新使用同一数据库事务。
- 文章公开详情和编辑详情返回原始正文及属于当前文章的图片 ID—URL 映射，由调用方完成展示。
- 文章软删除保留图片关系；硬删除按 `article_id` 查询并清理图片对象和记录，清理失败时保留文章以便重试。
- **BREAKING** 删除初始化空文章草稿接口，创建文章不再要求预先取得文章 ID。
- **BREAKING** 删除 `POST /article/image/upload-urls`，替换为 `POST /article/image/upload-url`。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `article-image-management`: 将基于文章目录的批量上传流程改为独立图片记录、单图片实时上传、正文占位符、文章关系同步和按关系清理。
- `article-lifecycle`: 移除为图片上传而初始化空草稿的要求，并调整文章创建、编辑和硬删除中的图片关系行为。

## Impact

- 影响 Article HTTP 路由、请求和响应 DTO、Application Service、Repository、事务协调、对象存储 Port 及模块组装。
- 新增 `article_images` 数据表，其中 `article_id` 可为空并建立普通索引，`object_key` 建立唯一索引。
- 创建和更新文章正文的存储语义从完整图片 URL 变更为 `image://<image_id>`。
- 前端编辑器需要区分持久化正文与图片展示 URL，并根据详情接口的图片映射渲染。
- 本次不实现 `article_id IS NULL` 图片的定时清理；上传未使用和编辑后解绑的图片由后续变更处理。
- 不新增第三方依赖；继续使用现有 MinIO、Goldmark、公开域名配置和本地事务协调器。
