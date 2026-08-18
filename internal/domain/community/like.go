package community

// 点赞状态取值
const (
	LikeStatusLiked    int8 = 1 // 点赞状态：已点赞
	LikeStatusCanceled int8 = 2 // 点赞状态：已取消点赞
)

// LikeTarget 是点赞目标类型（值对象），用于区分文章点赞与评论点赞。
type LikeTarget string

// 点赞目标类型取值
const (
	LikeTargetArticle LikeTarget = "article" // 点赞目标：文章
	LikeTargetComment LikeTarget = "comment" // 点赞目标：评论
)
