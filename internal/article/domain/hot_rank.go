package domain

// HotRankItem 是热榜条目（值对象），承载排行榜展示所需的文章统计快照。
type HotRankItem struct {
	ArticleID    uint64  // 文章ID
	Title        string  // 文章标题
	Hot          float64 // 热度值，由 CalcHotScore 计算得出
	ViewCount    uint32  // 浏览量
	CommentCount uint32  // 评论数
	LikeCount    uint32  // 点赞数
}

// CalcHotScore 按当前公式计算文章热度。
func CalcHotScore(viewCount, likeCount, commentCount uint32) float64 {
	return float64(viewCount + likeCount + commentCount)
}
