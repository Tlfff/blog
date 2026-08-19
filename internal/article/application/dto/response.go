package article

import (
	domain "blog/internal/article/domain"
	"blog/pkg/util/ip"
	"strings"
)

// ArticleDetailResponse 是文章详情响应 DTO。
type ArticleDetailResponse struct {
	ID           uint64   `json:"id"`            // 文章唯一标识
	Title        string   `json:"title"`         // 文章标题
	Content      string   `json:"content"`       // 文章正文内容（支持Markdown）
	Tags         []string `json:"tags"`          // 文章标签列表
	Status       int8     `json:"status"`        // 文章状态：1-已删除 2-草稿 3-已发表
	AuthorNick   string   `json:"author_nick"`   // 作者昵称
	AuthorAvatar string   `json:"author_avatar"` // 作者头像URL
	IP           string   `json:"ip"`            // 作者IP归属地（已由IP转换为地区文案）
	CreatedTime  int64    `json:"created_time"`  // 创建时间（Unix 秒）
	UpdatedTime  int64    `json:"updated_time"`  // 最后更新时间（Unix 秒）
	IsLiked      bool     `json:"is_liked"`      // 当前登录用户是否已点赞
	LikeCount    uint64   `json:"like_count"`    // 点赞数
}

// 构造单条详情响应
func NewArticleDetailResponse(m *domain.Article, nickName, avatar, authorIP string, isLiked bool) *ArticleDetailResponse {
	// 1. 模型为空时直接返回 nil，避免上层解引用空指针
	if m == nil {
		return nil
	}
	// 2. 标签字符串按英文逗号拆分为切片，无标签时返回空切片
	tags := strings.Split(m.Tags, ",")
	if m.Tags == "" {
		tags = []string{}
	}
	// 3. 组装响应：时间转为 Unix 秒，作者IP转换为地区文案
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
// ArticleListItem 是前台文章列表项 DTO。
type ArticleListItem struct {
	ID           uint64 `json:"id"`            // 文章ID
	Title        string `json:"title"`         // 文章标题
	Summary      string `json:"summary"`       // 正文摘要，取正文前50个字符
	AuthorID     uint64 `json:"author_id"`     // 作者用户ID
	UpdatedTime  int64  `json:"updated_time"`  // 最后更新时间（Unix 秒）
	ViewCount    uint32 `json:"view_count"`    // 浏览量
	LikeCount    uint32 `json:"like_count"`    // 点赞数
	CommentCount uint32 `json:"comment_count"` // 评论数
}

// ArticleListResponse 是前台文章列表响应 DTO。
type ArticleListResponse struct {
	List     []*ArticleListItem `json:"list"`      // 文章列表项集合
	LastID   uint64             `json:"last_id"`   // 游标锚点，作为下一页查询起点
	Total    uint64             `json:"total"`     // 总条数，仅走 Offset 分页时返回
	Page     uint64             `json:"page"`      // 当前页码
	PageSize uint64             `json:"page_size"` // 每页数量
}

// 构造列表响应
func NewArticleListResponse(models []*domain.Article, total, lastID, page, page_size uint64) *ArticleListResponse {
	// 1. 初始化响应骨架并写入分页信息
	resp := &ArticleListResponse{
		List:     make([]*ArticleListItem, 0),
		Total:    total,
		LastID:   lastID,
		Page:     page,
		PageSize: page_size,
	}

	// 2. 逐条把文章模型转换为列表项
	for _, m := range models {
		// 2.1 截取正文前50个字符作为摘要
		summary := m.Content
		contentRune := []rune(m.Content) // 转为字符切片
		if len(contentRune) > 50 {       // 如果超过50个字
			summary = string(contentRune[:50]) + "..." // 截取前50个字并转回字符串
		}

		// 2.2 追加列表项到响应
		resp.List = append(resp.List, &ArticleListItem{
			ID:           m.ID,
			Title:        m.Title,
			Summary:      summary,
			AuthorID:     m.AuthorID,
			UpdatedTime:  m.UpdatedTime.Unix(),
			CommentCount: m.CommentCount,
			ViewCount:    m.ViewCount,
			LikeCount:    m.LikeCount,
		})
	}
	// 3. 返回组装完成的列表响应
	return resp
}

// ====================  后台文章列表返回  ====================
// // AdminListItem 是后台文章列表项 DTO。
type AdminListItem struct {
	ID          uint64   `json:"id"`           // 文章ID
	Title       string   `json:"title"`        // 文章标题
	Tags        []string `json:"tags"`         // 文章标签列表
	Status      int8     `json:"status"`       // 文章状态：1-已删除 2-草稿 3-已发表
	CreatedTime int64    `json:"created_time"` // 创建时间（Unix 秒）
	UpdatedTime int64    `json:"updated_time"` // 最后更新时间（Unix 秒）
}

// // AdminListResponse 是后台文章列表响应 DTO。
type AdminListResponse struct {
	List     []*AdminListItem `json:"list"`      // 文章列表项集合
	LastID   uint64           `json:"last_id"`   // 游标锚点，作为下一页查询起点
	Total    uint64           `json:"total"`     // 总条数，仅走 Offset 分页时返回
	Page     uint64           `json:"page"`      // 当前页码
	PageSize uint64           `json:"page_size"` // 每页数量
}

// 构建后台列表
func NewAdminListResponse(models []*domain.Article, total, lastID, page, page_size uint64) *AdminListResponse {
	// 1. 初始化响应骨架并写入分页信息
	resp := &AdminListResponse{
		List:     make([]*AdminListItem, 0),
		Total:    total,
		LastID:   lastID,
		Page:     page,
		PageSize: page_size,
	}

	// 2. 逐条把文章模型转换为后台列表项
	for _, m := range models {
		// 2.1 标签字符串按英文逗号拆分，无标签时返回空切片
		tags := strings.Split(m.Tags, ",")
		if m.Tags == "" {
			tags = []string{}
		}
		// 2.2 追加列表项到响应
		resp.List = append(resp.List, &AdminListItem{
			ID:          m.ID,
			Title:       m.Title,
			Tags:        tags,
			Status:      m.Status,
			CreatedTime: m.CreatedTime.Unix(),
			UpdatedTime: m.UpdatedTime.Unix(),
		})
	}
	// 3. 返回组装完成的列表响应
	return resp
}

// ====================  文章排行榜列表返回  ====================
// // HotRankResponse 是文章热榜响应 DTO。
type HotRankResponse struct {
	List *[]HotRankItem `json:"list"` // 热榜条目集合
}

// // HotRankItem 是文章热榜条目 DTO。
type HotRankItem struct {
	ArticleID    uint64  `json:"article_id"`    // 文章id
	Title        string  `json:"title"`         // 文章标题
	Hot          float64 `json:"hot"`           // 热度
	ViewCount    uint32  `json:"view_count"`    // 浏览量
	CommentCount uint32  `json:"comment_count"` // 评论数
	LikeCount    uint32  `json:"like_count"`    // 点赞数
}

// 构造文章热榜响应
func NewHotRankResponse(list []HotRankItem) *HotRankResponse {
	resp := &HotRankResponse{
		List: &list,
	}
	return resp
}

// --------------------------------------- 二方服务 ---------------------------------------
// // ExternalListItem 是开放接口文章列表项 DTO（对二方/三方合作方暴露）。
type ExternalListItem struct {
	ID          uint64   `json:"id"`           // 文章ID
	Title       string   `json:"title"`        // 文章标题
	Tags        []string `json:"tags"`         // 文章标签列表
	CreatedTime int64    `json:"created_time"` // 创建时间（Unix 秒）
	UpdatedTime int64    `json:"updated_time"` // 最后更新时间（Unix 秒）
}

// // ExternalListResponse 是开放接口文章列表响应 DTO。
type ExternalListResponse struct {
	List     []*ExternalListItem `json:"list"`      // 文章列表项集合
	Total    uint64              `json:"total"`     // 总文章数
	Page     uint64              `json:"page"`      // 当前页码
	PageSize uint64              `json:"page_size"` // 每页数量
}

// 构建已发表文章列表响应
func NewExternalListResponse(models []*domain.Article, total, page, page_size uint64) *ExternalListResponse {
	// 1. 初始化响应骨架并写入分页信息
	resp := &ExternalListResponse{
		List:     make([]*ExternalListItem, 0),
		Total:    total,
		Page:     page,
		PageSize: page_size,
	}

	// 2. 逐条把文章模型转换为开放列表项
	for _, m := range models {
		// 2.1 标签字符串按英文逗号拆分，无标签时返回空切片
		tags := strings.Split(m.Tags, ",")
		if m.Tags == "" {
			tags = []string{}
		}
		// 2.2 追加列表项到响应
		resp.List = append(resp.List, &ExternalListItem{
			ID:          m.ID,
			Title:       m.Title,
			Tags:        tags,
			CreatedTime: m.CreatedTime.Unix(),
			UpdatedTime: m.UpdatedTime.Unix(),
		})
	}
	// 3. 返回组装完成的列表响应
	return resp
}
