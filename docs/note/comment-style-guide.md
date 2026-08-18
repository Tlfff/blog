# 注释规范（Comment Style Guide）

> 适用项目：blog-system/blog（Go + gRPC/Protobuf）
> 适用对象：Go 代码（结构体、函数、接口、常量等）与 `proto/*.proto` 契约文件
> 注释语言：统一使用简体中文；代码标识符、包名、字段名保持英文

## 1. 总体原则

1. 注释解释"做了什么 / 为什么这样做"，不逐行复述代码本身。
2. 注释统一使用 `//` 单行风格，不使用 `/* */` 块注释。
3. 术语（聚合根、实体、值对象、DTO、RPC）可保留英文原词。
4. 注释必须随代码一起维护：改动逻辑时同步更新相关注释，禁止留下过时注释。
5. 所有注释应帮助读者快速理解：结构体归属、字段含义、函数职责、执行流程、枚举取值。

## 2. 结构体注释

### 2.1 类型注释（聚合根 / 实体 / 值对象）

- 结构体上方必须有一行注释，说明它属于哪个领域、扮演什么角色。
- 格式：`// Xxx 是 <领域> 领域的<角色>。`
- 角色统一使用：聚合根、聚合、实体、值对象、DTO、查询模型、事件 等。
- 无法确定领域归属的通用类型，直接说明其用途即可。

示例：

```go
// Article 是 Content 领域的聚合根。
type Article struct { ... }

// Comment 是 Community 领域的评论聚合。
type Comment struct { ... }

// User 是 Identity 领域的聚合根。
type User struct { ... }

// Session 表示一次登录会话（值对象）。
type Session struct { ... }
```

### 2.2 字段注释

- 每个字段后跟行尾注释 `// 说明`，与代码之间至少一个空格。
- 同一结构体内多个字段的注释尽量按最长的一列对齐，保持视觉整齐。
- 字段注释必须说明业务含义；枚举/状态类字段要写明取值范围及含义。
- 主键、时间等通用字段也要注释（如"创建时间""最后更新时间"），不允许留空。

model 层（GORM 映射）示例：

```go
// Article 是文章数据模型（GORM 映射）。
type Article struct {
	ID           uint64    `gorm:"column:id;primaryKey;autoIncrement" ` // 文章唯一标识
	AuthorID     uint64    `gorm:"column:author_id" `                   // 作者用户ID
	Title        string    `gorm:"column:title" `                       // 文章标题
	Content      string    `gorm:"column:content" `                     // 文章正文内容（支持Markdown）
	Tags         string    `gorm:"column:tags" `                        // 文章标签
	Status       int8      `gorm:"column:status" `                      // 文章状态：1-已删除 2-草稿 3-已发表
	ViewCount    uint32    `gorm:"column:view_count" `                  // 浏览量
	LikeCount    uint32    `gorm:"column:like_count"`                   // 点赞数
	CommentCount uint32    `gorm:"column:comment_count" `               // 评论数
	CreatedTime  time.Time `gorm:"column:created_time;autoCreateTime" ` // 创建时间
	UpdatedTime  time.Time `gorm:"column:updated_time;autoUpdateTime" ` // 最后更新时间
}
```

domain 层示例：

```go
// Article 是 Content 领域的聚合根。
type Article struct {
	ID           uint64    // 文章唯一标识
	AuthorID     uint64    // 作者用户ID
	Title        string    // 文章标题
	Content      string    // 文章正文内容（支持Markdown）
	Tags         string    // 文章标签
	Status       int8      // 文章状态：1-已删除 2-草稿 3-已发表
	ViewCount    uint32    // 浏览量
	LikeCount    uint32    // 点赞数
	CommentCount uint32    // 评论数
	CreatedTime  time.Time // 创建时间
	UpdatedTime  time.Time // 最后更新时间
}
```

规则：

1. 不允许存在没有任何注释的结构体字段（测试/临时结构体除外，但建议同样加注释）。
2. 状态类字段必须列出所有取值及含义，如 `// 文章状态：1-已删除 2-草稿 3-已发表`。
3. 加密/算法类字段要说明存储格式，如 `// PBKDF2加密后的密码：算法$迭代次数$Salt$Hash`。

> 提示：Go 的 `gofmt` 不会自动对齐行尾注释，建议在编辑器中开启注释对齐（如 GoLand：Settings > Editor > Code Style > Go > Other > Align comments），或手动对齐。

## 3. 函数与方法注释

### 3.1 函数前注释（职责说明）

- 每个函数/方法上方加一行注释，说明这个函数是干什么的。
- **不写函数名**，直接用动词短语描述职责，如 `// 创建文章`、`// 校验密码`、`// 获取图片上传凭证`。
- 若行为不易从名字看出，补充说明参数含义、返回值、错误、副作用或注意事项。
- 注释以 `//` 开头，动词开头，句末一般不加句号（描述性长句可加句号）。

示例：

```go
// 创建文章
func (h *ArticleHandler) CreateArticle(c *gin.Context) { ... }

// 校验密码与存储哈希是否匹配
func VerifyPassword(password, storedPassword string) (bool, error) { ... }
```

正反例：

```go
// 反例：重复函数名，等于没写
// CreateArticle 创建文章

// 正例：直接说明职责
// 创建文章
```

> 说明：Go 官方 golint 默认建议导出函数注释以函数名开头（`// CreateArticle 创建文章`），但本项目为可读性统一约定**不写函数名**，以动词短语直接描述职责。团队规范优先于该默认建议。

### 3.2 函数内流程注释（编号分步）

- 函数体内部，在关键步骤前用 `// 1. 描述`、`// 2. 描述` 标明执行流程。
- 二级流程用 `// 1.1 描述`、`// 1.2 描述`，**最多两级，禁止出现 1.1.1 这种三级编号**。
- 编号注释与所在代码块缩进一致，放在对应语句上方。
- 分支/错误处理中的关键判断也可加编号注释说明目的。

示例：

```go
// 创建文章
func (h *ArticleHandler) CreateArticle(c *gin.Context) {
	var req arcticleDto.CreateArticleRequest
	// 1. 解析请求体并放进req
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.ErrInvalidRequestBody)
		return
	}

	// 2. 从上下文中获取用户信息，MustGet表示一定会有数据返回，所以只返回any，Get会返回bool和any
	user := c.MustGet("currentUser").(*auth.UserContext)
	// 3. 调用service创建文章
	err := h.article.CreateArticle(
		c,
		uint64(user.UserID),
		req.Title,
		req.Content,
		req.Tags,
		req.Status,
	)
	if err != nil {
		c.Error(err)
		return
	}
	// 4. 返回成功响应
	common.OK(c, "文章创建成功", nil)
}
```

二级编号示例：

```go
// 生成 PBKDF2 密码哈希
func HashPassword(password string) (string, error) {
	// 1. 生成随机盐值
	salt := make([]byte, SaltLength)
	// ...
	// 2. 计算哈希
	// 2.1 使用 PBKDF2 派生密钥
	hash, err := pbkdf2.Key(sha256.New, password, salt, HashIterations, HashKeyLength)
	if err != nil {
		return "", err
	}
	// 2.2 编码为十六进制字符串
	hashStr := hex.EncodeToString(hash)
	// 3. 拼装最终存储格式：算法$迭代次数$Salt$Hash
	return fmt.Sprintf("%s$%d$%s$%s", PasswordHashAlgorithm, HashIterations, saltStr, hashStr), nil
}
```

## 4. proto 文件注释

proto 文件是跨服务契约，注释要详尽，覆盖文件、service、rpc、message、字段、枚举。

### 4.1 文件级注释

- 文件头部说明文件用途与所属服务/模块。
- `syntax`、`package`、`option go_package` 建议加行尾注释说明，或在文件头统一说明。

示例：

```proto
// article.proto：Content 领域文章服务的开放接口定义。
syntax = "proto3";

package blogopen.v1;

option go_package = "blog/gen/article;article"; // Go 导入路径为 blog/gen/article，Go 包名为 article
```

### 4.2 service 与 rpc 注释

- service 上方注释说明服务职责。
- 每个 rpc 上方注释说明接口功能、请求/响应要点，必要时说明使用方。

示例：

```proto
// 文章服务定义：提供文章查询与互动的开放接口。
service ArticleService {
  // 获取全部可用文章列表（除已删除）
  rpc GetAvailableList (GetExternalListRequest) returns (ExternalListResponse);
}
```

### 4.3 message 注释

- 每个 message 上方注释说明它是请求还是响应，以及用途，格式：`// 请求：...` / `// 响应：...`。

示例：

```proto
// 请求：获取文章列表
message GetExternalListRequest { ... }

// 响应：文章列表
message ExternalListResponse { ... }
```

### 4.4 字段注释

- 每个字段加行尾注释说明含义。
- 枚举/状态类字段（int32 等）必须写明取值范围与含义；时间戳注明单位（秒/毫秒）。
- 多字段时按最长注释对齐。

示例：

```proto
// 响应：文章列表
message ExternalListResponse {
  repeated ArticleItem items = 1; // 文章列表
  uint64 total = 2;               // 总文章数
  uint64 page = 3;                // 当前页码
  uint64 page_size = 4;           // 每页数量
}
```

### 4.5 enum 注释

- enum 上方说明用途，每个枚举值加行尾注释。

示例：

```proto
// 文章状态
enum ArticleStatus {
  ARTICLE_STATUS_UNSPECIFIED = 0; // 未指定
  ARTICLE_STATUS_DRAFT = 1;       // 草稿
  ARTICLE_STATUS_PUBLISHED = 2;   // 已发表
}
```

## 5. 其他注释场景

| 场景 | 要求 | 示例 |
| --- | --- | --- |
| 常量 | 说明含义/取值 | `StatusPublished int8 = 3 // 已发表` |
| 接口 | 说明职责 | `// ArticleUsecase 是文章生命周期、查询和图片能力的应用用例接口。` |
| 包 | 说明包用途 | `// Package content 提供 Content 领域的应用用例。` |
| TODO/FIXME | 说明待办内容与原因 | `// TODO: 后续迁移到独立鉴权服务` |
| 错误定义 | 说明含义与触发场景 | `// ErrArticleNotFound 文章不存在` |

## 6. 自检清单

提交代码前逐项检查：

1. 每个结构体上方有类型注释（聚合根/实体/值对象/用途）？
2. 每个字段有行尾注释？状态类字段写全取值含义？
3. 每个函数上方有职责注释且不含函数名？
4. 函数内关键流程有 `// 1. 2. 3.` 编号注释？二级最多到 `1.1`，无三级编号？
5. proto 文件里 service / rpc / message / 字段 / enum 都有注释？
6. 注释与代码同步，无过时或重复注释？
