package community

const (
	LikeStatusLiked   int8 = 1
	LikeStatusCanceled int8 = 2
)

// LikeTarget 区分文章点赞与评论点赞。
type LikeTarget string

const (
	LikeTargetArticle LikeTarget = "article"
	LikeTargetComment LikeTarget = "comment"
)
