package consts

const (
	KeyArticleLikeStatePrefix = "like:article:status:" // 用户点赞文章操作hash
	KeyCommentLikeStatePrefix = "like:comment:status:" // 用户点赞评论操作hash

	KeyArticleLikeLockPrefix = "lock:init:like:article:" // 初始化用户点赞文章hash分布式锁Key
	KeyCommentLikeLockPrefix = "lock:init:like:comment:" // 初始化用户点赞评论hash分布式锁Key
	KeyArticleInfoLockPrefix = "lock:init:article:info:" // 初始化文章info hash分布式锁Key

	KeyCommentLikeCountPrefix = "like:comment:count:" // 评论点赞计数Key todo： 删除

	KeyArticleHotRankZSet     = "rank:article:hot"           // 全局热度排行榜ZSet
	KeyArticleInfoHashPrefix  = "article:info:"              // 单篇文章信息Hash前缀
	KeyArticleHotRankInitLock = "lock:init:rank:article:hot" // 排行榜初始化全局锁

	FeildArticleLikeCount    = "like_count"    // 文章info哈希的field
	FeildArticleViewCount    = "view_count"    // 文章info哈希的field
	FeildArticleCommentCount = "comment_count" // 文章info哈希的field

	// ----------------------------- 重构后的key -----------------------------
	// ----------------------------- 文章点赞相关key -----------------------------
	KeyLikeArticlePre  = "like:article:"      // set,记录用户是否点赞文章
	KeyLockLikeArticle = "lock:like:article:" // 锁，在冷启动缓存时使用
	// ----------------------------- 评论点赞相关key -----------------------------
	KeyLikeCommentPre  = "like:comment:"      // set,记录用户是否点赞文章
	KeyLockLikeComment = "lock:like:comment:" // 锁，在冷启动缓存时使用

)
