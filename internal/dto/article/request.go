package article

// 创建文章
type CreateArticleRequest struct {
	Title   string   `json:"title" binding:"required"` //不能为空
	Content string   `json:"content" binding:"required"`
	Tags    []string `json:"tags"`
	Status  int8     `json:"status" binding:"omitempty,oneof=2 3"`
}

// 修改文章
type UpdateArticleRequest struct {
	ID      uint64   `json:"id" binding:"required,min=0"` // id是否大于0
	Title   string   `json:"title" binding:"required"`
	Content string   `json:"content" binding:"required"`
	Tags    []string `json:"tags"`
	Status  int8     `json:"status" binding:"omitempty,oneof=2 3"` // 状态只能是2,3
}

// 删除文章
type DeleteArticleRequest struct {
	ID uint64 `json:"id" binding:"min=0"`
}

// 发布文章
type PublishArticleRequest struct {
	ID uint64 `json:"id" binding:"required,min=0"`
}

// 获取文章详情
type GetDetailRequest struct {
	ID uint64 `form:"id" binding:"required,min=0"` // form:"id" 告诉 Gin 去 URL 参数中找 ?id=xxx
}

// 获取用户文章列表
type GetPublishListRequest struct {
	LastID   uint64 `form:"last_id" binding:"omitempty,min=0"`
	Page     uint64 `form:"page" binding:"omitempty,min=0"`
	PageSize uint64 `form:"page_size" binding:"min=10,max=20"`
	IsDesc   bool   `form:"is_desc"` // 是否按时间倒序（默认false正序）
}

// 管理者：获取文章列表
type GetAdminListRequest struct {
	Status   int8   `form:"status" binding:"required,min=0"`
	LastID   uint64 `form:"last_id" binding:"omitempty,min=0"`
	Page     uint64 `form:"page" binding:"omitempty,min=0"`
	PageSize uint64 `form:"page_size" binding:"min=10,max=20"`
	IsDesc   bool   `form:"is_desc"` // 是否按时间倒序（默认false正序）
}

// 恢复文章
type RecoverArticleRequest struct {
	ID uint64 `json:"id" binding:"required,min=0"`
}
