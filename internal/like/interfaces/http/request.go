package http

// ArticleIdRequest 是文章点赞与取消点赞的请求 DTO。
type ArticleIdRequest struct {
	ArticleID uint64 `json:"article_id" binding:"required"` // 目标文章ID，不能为空
}

// CommentIdRequest 是评论点赞与取消点赞的请求 DTO。
type CommentIdRequest struct {
	CommentID uint64 `json:"comment_id" binding:"required"` // 目标评论ID，不能为空
}
