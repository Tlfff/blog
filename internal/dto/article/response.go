package article

import (
	"blog/internal/model"
	"time"
)

// 文章详情
type ArticleDetailResponse struct {
	ID         int64     `json:"id"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	Tags       []string  `json:"tags"`
	Status     int8      `json:"status"`
	AuthorID   int64     `json:"author_id"`
	AddTime    time.Time `json:"Add_time"`
	UpdateTime time.Time `json:"update_time"`
}

// 列表项
type ArticleListItem struct {
	ID         int64     `json:"id"`
	Title      string    `json:"title"`
	Summary    string    `json:"summary"` // 摘要
	AuthorID   int64     `json:"author_id"`
	UpdateTime time.Time `json:"update_time"`
	// LikeCount    int64 `json:"like_count"`
	// CommentCount int64 `json:"comment_count"`
	// Heat int64 `json:"heat"`
}

// 列表返回
type ArticleListResponse struct {
	List  []*ArticleListItem `json:"list"`
	Total int                `json:"total"`
}

// 构造单条详情响应
func NewArticleDetailResponse(m *model.Article) *ArticleDetailResponse {
	if m == nil {
		return nil
	}
	return &ArticleDetailResponse{
		ID:         m.ID,
		Title:      m.Title,
		Content:    m.Content,
		Tags:       m.Tags,
		Status:     int8(m.Status),
		AuthorID:   m.AuthorID,
		AddTime:    m.AddTime,
		UpdateTime: m.UpdateTime,
	}
}

// 构造列表响应
func NewArticleListResponse(models []*model.Article) *ArticleListResponse {
	resp := &ArticleListResponse{
		List:  make([]*ArticleListItem, 0),
		Total: 0,
	}

	for _, m := range models {
		summary := m.Content
		contentRune := []rune(m.Content) // 转为字符切片
		if len(contentRune) > 50 {       // 如果超过50个字
			summary = string(contentRune[:50]) + "..." // 截取前50个字并转回字符串
		}

		resp.List = append(resp.List, &ArticleListItem{
			ID:         m.ID,
			Title:      m.Title,
			Summary:    summary,
			AuthorID:   m.AuthorID,
			UpdateTime: m.UpdateTime,
		})
	}
	resp.Total = len(resp.List)
	return resp
}
