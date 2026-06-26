package article

// 创建文章
type CreateArticleRequest struct {
	ID      int64    `json:"id"`                       // 加入数据库后删除
	Title   string   `json:"title" binding:"required"` //不能为空
	Content string   `json:"content" binding:"required"`
	Tags    []string `json:"tags"`
	Status  int8     `json:"status"`
}

// 修改文章
type UpdateArticleRequest struct {
	ID      int64    `json:"id" binding:"required,min=0"` // id是否大于0
	Title   string   `json:"title" binding:"required"`
	Content string   `json:"content" binding:"required"`
	Tags    []string `json:"tags"`
	Status  int8     `json:"status" binding:"oneof=1 2"` // 状态只能是0,1,2
}

// 删除文章
type DeleteArticleRequest struct {
	ID int64 `json:"id" binding:"min=0"`
}

// 发布文章
type PublishArticleRequest struct {
	ID int64 `json:"id" binding:"required,min=0"`
}

// 获取文章详情
type GetDetailRequest struct {
	ID int64 `form:"id" binding:"required,min=0"` // form:"id" 告诉 Gin 去 URL 参数中找 ?id=xxx
}

// 获取用户文章列表
type GetPublishListRequest struct {
	AuthorID int64 `form:"author_id" binding:"required,min=1"`
}

// 管理者：获取文章列表
type GetAdminListRequest struct {
	Status int8 `form:"status" binding:"required,min=0"`
}

// 恢复文章
type RecoverArticleRequest struct {
	ID int64 `json:"id" binding:"required,min=0"`
}
