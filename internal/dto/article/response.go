package article

import (
	"blog/internal/model"
	"blog/pkg/util/ip"
	"strings"
)

// 文章详情
type ArticleDetailResponse struct {
	ID           uint64   `json:"id"`
	Title        string   `json:"title"`
	Content      string   `json:"content"`
	Tags         []string `json:"tags"`
	Status       int8     `json:"status"`
	AuthorNick   string   `json:"author_nick"`
	AuthorAvatar string   `json:"author_avatar"`
	IP           string   `json:"ip"` //作者IP
	CreatedTime  int64    `json:"created_time"`
	UpdatedTime  int64    `json:"updated_time"`
	IsLiked      bool     `json:"is_liked"`   // 是否点赞
	LikeCount    uint64   `json:"like_count"` // 点赞数量
}

// 构造单条详情响应
func NewArticleDetailResponse(m *model.Article, nickName, avatar, authorIP string, isLiked bool) *ArticleDetailResponse {
	if m == nil {
		return nil
	}
	tags := strings.Split(m.Tags, ",")
	if m.Tags == "" {
		tags = []string{}
	}
	return &ArticleDetailResponse{
		ID:           m.ID,
		Title:        m.Title,
		Content:      m.Content,
		Tags:         tags,
		Status:       int8(m.Status),
		AuthorNick:   nickName,
		AuthorAvatar: avatar,
		IP:           ip.ConvertIPToRegion(authorIP),
		CreatedTime:  m.CreatedTime.Unix(),
		UpdatedTime:  m.UpdatedTime.Unix(),
		IsLiked:      isLiked,
		LikeCount:    uint64(m.LikeCount),
	}
}

// ====================  前台文章列表返回  ====================
// 列表项
type ArticleListItem struct {
	ID          uint64 `json:"id"`
	Title       string `json:"title"`
	Summary     string `json:"summary"` // 摘要
	AuthorID    uint64 `json:"author_id"`
	UpdatedTime int64  `json:"updated_time"`
}

// 列表返回
type ArticleListResponse struct {
	List     []*ArticleListItem `json:"list"`
	LastID   uint64             `json:"last_id"`
	Total    uint64             `json:"total"`
	Page     uint64             `json:"page"`      // 页码
	PageSize uint64             `json:"page_size"` // 页面大小
}

// 构造列表响应
func NewArticleListResponse(models []*model.Article, total, lastID, page, page_size uint64) *ArticleListResponse {
	resp := &ArticleListResponse{
		List:     make([]*ArticleListItem, 0),
		Total:    total,
		LastID:   lastID,
		Page:     page,
		PageSize: page_size,
	}

	for _, m := range models {
		summary := m.Content
		contentRune := []rune(m.Content) // 转为字符切片
		if len(contentRune) > 50 {       // 如果超过50个字
			summary = string(contentRune[:50]) + "..." // 截取前50个字并转回字符串
		}

		resp.List = append(resp.List, &ArticleListItem{
			ID:          m.ID,
			Title:       m.Title,
			Summary:     summary,
			AuthorID:    m.AuthorID,
			UpdatedTime: m.UpdatedTime.Unix(),
		})
	}
	return resp
}

// ====================  后台文章列表返回  ====================
type AdminListItem struct {
	ID          uint64   `json:"id"`
	Title       string   `json:"title"`
	Tags        []string `json:"tags"`
	Status      int8     `json:"status"` // 状态：1所有，2草稿，3发布，0垃圾箱
	CreatedTime int64    `json:"created_time"`
	UpdatedTime int64    `json:"updated_time"`
}
type AdminListResponse struct {
	List     []*AdminListItem `json:"list"`
	LastID   uint64           `json:"last_id"`
	Total    uint64           `json:"total"`
	Page     uint64           `json:"page"`      // 页码
	PageSize uint64           `json:"page_size"` // 页面大小
}

// 构建后台列表
func NewAdminListResponse(models []*model.Article, total, lastID, page, page_size uint64) *AdminListResponse {
	resp := &AdminListResponse{
		List:     make([]*AdminListItem, 0),
		Total:    total,
		LastID:   lastID,
		Page:     page,
		PageSize: page_size,
	}

	for _, m := range models {
		tags := strings.Split(m.Tags, ",")
		if m.Tags == "" {
			tags = []string{}
		}
		resp.List = append(resp.List, &AdminListItem{
			ID:          m.ID,
			Title:       m.Title,
			Tags:        tags,
			Status:      m.Status,
			CreatedTime: m.CreatedTime.Unix(),
			UpdatedTime: m.UpdatedTime.Unix(),
		})
	}
	return resp
}

// ====================  文章排行榜列表返回  ====================
type HotRankResponse struct {
	List *[]HotRankItem `json:"list"`
}
type HotRankItem struct {
	ArticleID    uint64  `json:"article_id"`    // 文章id
	Title        string  `json:"title"`         // 文章标题
	Hot          float64 `json:"hot"`           // 热度
	ViewCount    uint32  `json:"view_count"`    // 浏览量
	CommentCount uint32  `json:"comment_count"` // 评论数
	LikeCount    uint32  `json:"like_count"`    // 点赞数
}

func NewHotRankResponse(list []HotRankItem) *HotRankResponse {
	resp := &HotRankResponse{
		List: &list,
	}
	return resp
}
