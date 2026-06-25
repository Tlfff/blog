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
	AuthorNick string    `json:"author_nick"`
	AddTime    time.Time `json:"add_time"`
	UpdateTime time.Time `json:"update_time"`
}

// 构造单条详情响应
func NewArticleDetailResponse(m *model.Article, nickName string) *ArticleDetailResponse {
	if m == nil {
		return nil
	}
	return &ArticleDetailResponse{
		ID:         m.ID,
		Title:      m.Title,
		Content:    m.Content,
		Tags:       m.Tags,
		Status:     int8(m.Status),
		AuthorNick: nickName, //这里应该放作者
		AddTime:    m.AddTime,
		UpdateTime: m.UpdateTime,
	}
}

// ====================  前台文章列表返回  ====================
// 列表项
type ArticleListItem struct {
	ID         int64     `json:"id"`
	Title      string    `json:"title"`
	Summary    string    `json:"summary"` // 摘要
	AuthorID   int64     `json:"author_id"`
	UpdateTime time.Time `json:"update_time"`
}

// 列表返回
type ArticleListResponse struct {
	List  []*ArticleListItem `json:"list"`
	Total int                `json:"total"`
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

// ====================  后台文章列表返回  ====================
type AdminListItem struct {
	ID         int64     `json:"id"`
	Title      string    `json:"title"`
	Tags       []string  `json:"tags"`
	Status     int8      `json:"status"` // 状态：1所有，2草稿，3发布，0垃圾箱
	AddTime    time.Time `json:"add_time"`
	UpdateTime time.Time `json:"update_time"`
}
type AdminListResponse struct {
	List  []*AdminListItem `json:"list"`
	Total int              `json:"total"`
}

// 构建后台列表
func NewAdminListResponse(models []*model.Article) *AdminListResponse {
	resp := &AdminListResponse{
		List:  make([]*AdminListItem, 0),
		Total: len(models),
	}

	for _, m := range models {

		resp.List = append(resp.List, &AdminListItem{
			ID:         m.ID,
			Title:      m.Title,
			Tags:       m.Tags,
			Status:     m.Status,
			AddTime:    m.AddTime,
			UpdateTime: m.UpdateTime,
		})
	}
	return resp
}
