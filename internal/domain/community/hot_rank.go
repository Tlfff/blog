package community

// HotRankItem 是热榜条目（值对象），承载排行榜展示所需的文章统计快照。
type HotRankItem struct {
	ArticleID    uint64  // 文章ID
	Title        string  // 文章标题
	Hot          float64 // 热度值，由 CalcHotScore 计算得出
	ViewCount    uint32  // 浏览量
	CommentCount uint32  // 评论数
	LikeCount    uint32  // 点赞数
}

// 按当前热度公式计算文章热度，公式为浏览量、点赞数、评论数三者之和
func CalcHotScore(viewCount, likeCount, commentCount uint32) float64 {
	return float64(viewCount + likeCount + commentCount)
}
