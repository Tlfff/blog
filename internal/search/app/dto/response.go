// Package dto 定义 Search Application 对外返回的数据结构。
package dto

// ArticleSearchItem 表示前台文章搜索结果项。
type ArticleSearchItem struct {
	ID             uint64   `json:"id"`              // 文章唯一标识
	Title          string   `json:"title"`           // 文章原始标题
	TitleHighlight string   `json:"title_highlight"` // 原始标题高亮，仅拼音命中时为空
	Summary        string   `json:"summary"`         // 正文纯文本高亮摘要，正文未命中时为空
	Tags           []string `json:"tags"`            // 规范化标签列表
}

// ArticleSearchResponse 表示前台文章搜索分页响应。
type ArticleSearchResponse struct {
	List     []ArticleSearchItem `json:"list"`      // 当前页文章搜索结果
	Total    uint64              `json:"total"`     // 符合关键词的已发表文章总数
	Page     uint64              `json:"page"`      // 当前页码，从 1 开始
	PageSize uint64              `json:"page_size"` // 每页数量，范围 10 至 20
}
