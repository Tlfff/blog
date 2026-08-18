package community

// HotRankItem 是热榜条目。
type HotRankItem struct {
	ArticleID    uint64
	Title        string
	Hot          float64
	ViewCount    uint32
	CommentCount uint32
	LikeCount    uint32
}

// CalcHotScore 按当前热度公式计算文章热度。
func CalcHotScore(viewCount, likeCount, commentCount uint32) float64 {
	return float64(viewCount + likeCount + commentCount)
}
