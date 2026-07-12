package consts

const (
	// 文章点赞Hash前缀 (指定规范)
	KeyArticleLikeStatePrefix = "article:likes:status:"
	// 文章点赞计数Key
	KeyArticleLikeCountPrefix = "article:likes:count:"
	// 文章点赞分布式锁Key前缀
	KeyArticleLikeLockPrefix = "lock:article:like:"

	// 评论点赞状态Hash前缀 (供后续对齐)
	KeyCommentLikeStatePrefix = "like:comment:status:"
	// 评论点赞计数Key
	KeyCommentLikeCountPrefix = "like:comment:count:"
	KeyCommentLikeLockPrefix  = "lock:comment:like:"
)
