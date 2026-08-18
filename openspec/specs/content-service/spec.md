## Purpose

为博客系统提供文章内容的完整生命周期和内容资源管理能力；阶段一由单体内 Content 领域模块提供，阶段二由 Content Service 承载，确保文章状态、作者权限、公开查询和图片上传行为由文章内容边界统一负责。

## Requirements

### Requirement: Article lifecycle management

内容服务 SHALL 支持文章创建、编辑、发布、软删除、恢复和彻底删除，并 SHALL 按文章状态和作者/管理员权限限制操作。

#### Scenario: Administrator creates a draft
- **WHEN** 具有管理员权限的调用方提交合法标题、正文、标签和草稿状态
- **THEN** 系统创建属于该调用方的文章草稿，并返回成功结果

#### Scenario: Published article is publicly readable
- **WHEN** 游客或普通用户查询已发布文章
- **THEN** 系统返回文章公开内容；草稿、已删除文章不得通过公开查询返回

#### Scenario: Unauthorized lifecycle operation is rejected
- **WHEN** 非作者且非管理员的调用方尝试编辑、发布、删除或恢复文章
- **THEN** 系统拒绝操作并返回统一权限错误，文章状态保持不变

#### Scenario: Soft-deleted article can be recovered
- **WHEN** 管理员对垃圾箱中的文章执行恢复
- **THEN** 文章恢复为可管理状态，但不应在未发布前出现在公开文章列表中

### Requirement: Article query and external access

内容服务 SHALL 提供公开文章列表、后台分页列表、文章详情、垃圾箱列表和开放 API 所需的可用文章查询，并 SHALL 支持现有 offset/游标分页语义。

#### Scenario: Cursor pagination returns stable continuation
- **WHEN** 调用方使用合法的 `last_id` 和分页大小请求文章列表
- **THEN** 系统返回不重复且符合排序方向的下一页结果，并提供继续翻页所需的信息

#### Scenario: External list excludes deleted articles
- **WHEN** 合法的开放 API 调用方请求可用文章列表
- **THEN** 系统返回非删除文章，并保持现有 gRPC 响应字段和分页语义

### Requirement: Article media upload and promotion

内容服务 SHALL 为文章正文图片提供临时上传凭证，并 SHALL 在文章保存或发布时校验图片归属和对象路径，避免无效对象污染正式目录。

#### Scenario: Valid image upload URL is issued
- **WHEN** 管理员请求允许扩展名的文章图片上传凭证
- **THEN** 系统返回临时上传地址和可供正文引用的访问地址

#### Scenario: Invalid image extension is rejected
- **WHEN** 调用方请求不在配置白名单内的文件扩展名
- **THEN** 系统拒绝生成上传凭证

#### Scenario: Temporary images are promoted with the article
- **WHEN** 文章正文引用合法的临时图片并成功保存
- **THEN** 系统将图片转移到文章正式路径，并保存可访问的正文引用
