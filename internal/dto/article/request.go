package article

// 创建文章
type CreateArticleRequest struct {
	Title   string   `json:"title" binding:"required"` //不能为空
	Content string   `json:"content" binding:"required"`
	Tags    []string `json:"tags"`
	Status  int8     `json:"status"`
}

// 修改文章
type UpdateArticleRequest struct {
	ID      uint64   `json:"id" binding:"required,min=0"` // id是否大于0
	Title   string   `json:"title" binding:"required"`
	Content string   `json:"content" binding:"required"`
	Tags    []string `json:"tags"`
	Status  int8     `json:"status" binding:"oneof=1 2"` // 状态只能是0,1,2
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
	AuthorID uint64 `form:"author_id" binding:"required,min=1"`
}

// 管理者：获取文章列表
type GetAdminListRequest struct {
	Status int8 `form:"status" binding:"required,min=0"`
}

// 恢复文章
type RecoverArticleRequest struct {
	ID uint64 `json:"id" binding:"required,min=0"`
}
