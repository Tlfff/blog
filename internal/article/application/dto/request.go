package article

// CreateArticleRequest 是创建文章的请求 DTO。
type CreateArticleRequest struct {
	Title   string   `json:"title" binding:"required"`             // 文章标题，不能为空
	Content string   `json:"content" binding:"required"`           // 文章正文内容（支持Markdown），不能为空
	Tags    []string `json:"tags"`                                 // 文章标签列表，可为空
	Status  int8     `json:"status" binding:"omitempty,oneof=2 3"` // 创建后的状态：2-草稿 3-已发表，不传时由服务端取默认值
}

// UpdateArticleRequest 是修改文章的请求 DTO。
type UpdateArticleRequest struct {
	ID      uint64   `json:"id" binding:"required,min=0"`          // 待修改的文章ID，必须大于0
	Title   string   `json:"title" binding:"required"`             // 文章标题，不能为空
	Content string   `json:"content" binding:"required"`           // 文章正文内容（支持Markdown），不能为空
	Tags    []string `json:"tags"`                                 // 文章标签列表，可为空
	Status  int8     `json:"status" binding:"omitempty,oneof=2 3"` // 修改后的状态：只能是 2-草稿 3-已发表
}

// DeleteArticleRequest 是软删除文章的请求 DTO。
type DeleteArticleRequest struct {
	ID uint64 `json:"id" binding:"min=0"` // 待删除的文章ID
}

// PublishArticleRequest 是发布文章的请求 DTO。
type PublishArticleRequest struct {
	ID uint64 `json:"id" binding:"required,min=0"` // 待发布的文章ID，必须大于0
}

// GetDetailRequest 是获取文章详情的请求 DTO。
type GetDetailRequest struct {
	ID uint64 `form:"id" binding:"required,min=0"` // 文章ID；form:"id" 告诉 Gin 去 URL 参数中找 ?id=xxx
}

// GetPublishListRequest 是获取已发表文章列表的请求 DTO。
type GetPublishListRequest struct {
	LastID   uint64 `form:"last_id" binding:"omitempty,min=0"` // 游标ID，用于游标分页，传0表示从头开始
	Page     uint64 `form:"page" binding:"omitempty,min=0"`    // 传统页码，用于跳页
	PageSize uint64 `form:"page_size" binding:"min=10,max=20"` // 每页条数，取值范围 10~20
	IsDesc   bool   `form:"is_desc"`                           // 是否按时间倒序（默认false正序）
}

// GetAdminListRequest 是管理端按状态获取文章列表的请求 DTO。
type GetAdminListRequest struct {
	Status   int8   `form:"status" binding:"required"`         // 文章状态过滤：-2-全部 -1-除已删除 1-已删除 2-草稿 3-已发表
	LastID   uint64 `form:"last_id" binding:"omitempty,min=0"` // 游标ID，用于游标分页，传0表示从头开始
	Page     uint64 `form:"page" binding:"omitempty,min=0"`    // 传统页码，用于跳页
	PageSize uint64 `form:"page_size" binding:"min=10,max=20"` // 每页条数，取值范围 10~20
	IsDesc   bool   `form:"is_desc"`                           // 是否按时间倒序（默认false正序）
}

// RecoverArticleRequest 是恢复已删除文章的请求 DTO。
type RecoverArticleRequest struct {
	ID uint64 `json:"id" binding:"required,min=0"` // 待恢复的文章ID，必须大于0
}

// GetImageUploadURLRequest 是获取文章图片上传凭证的请求 DTO。
type GetImageUploadURLRequest struct {
	FileExt string `json:"file_ext" binding:"required"` // 图片文件扩展名，如 jpg/png，不能为空
}
