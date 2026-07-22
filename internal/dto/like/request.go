package like

type ArticleIdRequest struct {
	ArticleID uint64 `json:"article_id" binding:"required"`
}
type CommentIdRequest struct {
	CommentID uint64 `json:"comment_id" binding:"required"`
}
