package comment

// 创建评论（主评论或子评论通用）
type CreateCommentRequest struct {
	ArticleID     uint64 `json:"article_id" binding:"required,min=1"` // 文章ID
	RootID        uint64 `json:"root_id" binding:"min=0"`             // 0:主评论, 大于0:子评论
	ReplyToUserID uint64 `json:"reply_to_user_id" binding:"min=0"`    // 被回复者UID (子评论时可选)
	Content       string `json:"content" binding:"required"`          // 评论内容
}

// DeleteCommentRequest 用户删除自己的评论
type DeleteCommentRequest struct {
	ID uint64 `json:"id" binding:"required,min=1"` // 评论ID
}

// 获取文章主评论列表
type GetRootListRequest struct {
	ArticleID uint64 `form:"article_id" binding:"required,min=1"` // 文章ID
	Page      int    `form:"page" binding:"min=0"`                // 传统页码（跳页用）
	PageSize  int    `form:"page_size" binding:"min=10,max=20"`   // 每页条数(可以调整范围，默认10)
	LastID    uint64 `form:"last_id" binding:"min=0"`             // 游标ID
	IsDesc    bool   `form:"is_desc"`                             // 是否按时间倒序（默认false正序）
	AuthorID  uint64 `form:"author_id" binding:"min=0"`           // 只看楼主的用户ID（可选）
}

// 展开拉取子评论列表
type GetReplyListRequest struct {
	RootID   uint64 `form:"root_id" binding:"required,min=1"`  // 对应的主评论/楼层ID
	Page     int    `form:"page" binding:"min=0"`              // 传统页码（跳页用）
	PageSize int    `form:"page_size" binding:"min=10,max=20"` // 每页条数(可以调整范围，默认10)
	LastID   uint64 `form:"last_id" binding:"min=0"`           // 游标ID
}

// 管理员后台强删
type AdminDeleteRequest struct {
	ID uint64 `json:"id" binding:"required,min=1"`
}
