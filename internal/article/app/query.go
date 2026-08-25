package app

// ArticleDetailQuery 表示文章详情查询条件。
type ArticleDetailQuery struct {
	ArticleID uint64 // 文章唯一标识
	UserID    uint64 // 当前用户唯一标识，游客为 0
}

// ArticleListQuery 表示文章列表查询条件。
type ArticleListQuery struct {
	Status   int8   // 文章状态过滤条件
	LastID   uint64 // 游标文章唯一标识
	Page     uint64 // 页码
	PageSize uint64 // 每页数量
	IsDesc   bool   // 是否按唯一标识倒序
}
