package http

// ArticleSearchRequest 表示前台文章搜索查询参数。
type ArticleSearchRequest struct {
	Keyword  string `form:"keyword" binding:"required"`                 // 搜索关键词，去除首尾空格后不能为空
	Page     uint64 `form:"page" binding:"required,min=1"`              // 页码，从 1 开始
	PageSize uint64 `form:"page_size" binding:"required,min=10,max=20"` // 每页数量，范围 10 至 20
}
