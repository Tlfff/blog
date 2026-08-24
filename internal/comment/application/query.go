package application

// RootCommentListQuery 表示主评论列表查询条件。
type RootCommentListQuery struct {
	ArticleID uint64 // 文章唯一标识
	LastID    uint64 // 游标评论唯一标识
	Page      int    // 页码
	PageSize  int    // 每页数量
	IsDesc    bool   // 是否倒序
	AuthorID  uint64 // 只看楼主时的作者唯一标识
}

// ReplyListQuery 表示回复列表查询条件。
type ReplyListQuery struct {
	RootID   uint64 // 根评论唯一标识
	LastID   uint64 // 游标回复唯一标识
	Page     int    // 页码
	PageSize int    // 每页数量
}
