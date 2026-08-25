package app

// ArticleLikeStateQuery 表示文章点赞状态查询条件。
type ArticleLikeStateQuery struct {
	UserID    uint64 // 用户唯一标识
	ArticleID uint64 // 文章唯一标识
}
