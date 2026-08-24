package application

// CreateCommentCommand 表示创建评论或回复用例输入。
type CreateCommentCommand struct {
	ArticleID     uint64 // 所属文章唯一标识
	RootID        uint64 // 根评论唯一标识，主评论为 0
	UserID        uint64 // 评论用户唯一标识
	ReplyToUserID uint64 // 被回复用户唯一标识
	Content       string // 评论正文
	IP            string // 评论来源 IP
}

// DeleteCommentCommand 表示删除评论用例输入。
type DeleteCommentCommand struct {
	CommentID uint64 // 评论唯一标识
	UserID    uint64 // 操作用户唯一标识
	IsAdmin   bool   // 是否管理员操作
}
