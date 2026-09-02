package http

import (
	"blog/internal/platform/interfaces/http/response"
	searchdto "blog/internal/search/app/dto"
	searchdomain "blog/internal/search/domain"
	apperrors "blog/internal/shared/apperrors"
	"context"
	"errors"

	"github.com/gin-gonic/gin"
)

// SearchUsecase 定义前台文章搜索应用能力。
type SearchUsecase interface {
	// SearchArticles 按关键词和页码查询已发表文章。
	SearchArticles(ctx context.Context, keyword string, page, pageSize uint64) (*searchdto.ArticleSearchResponse, error)
}

// Handler 提供前台文章搜索 HTTP 接口。
type Handler struct {
	search SearchUsecase // 文章搜索应用用例
}

// NewHandler 创建前台文章搜索 HTTP Handler。
func NewHandler(search SearchUsecase) *Handler {
	// 1. 保存搜索应用用例
	return &Handler{search: search}
}

// SearchArticles 查询已发表文章并返回高亮分页结果。
func (h *Handler) SearchArticles(c *gin.Context) {
	// 1. 绑定并校验固定搜索参数
	var request ArticleSearchRequest
	if err := c.ShouldBindQuery(&request); err != nil {
		_ = c.Error(apperrors.ErrParameter)
		return
	}

	// 2. 调用 Search Application 执行查询
	result, err := h.search.SearchArticles(c.Request.Context(), request.Keyword, request.Page, request.PageSize)
	if err != nil {
		_ = c.Error(mapSearchError(err))
		return
	}

	// 3. 返回稳定搜索响应契约
	response.OK(c, "搜索成功", result)
}

// mapSearchError 把 Search 领域错误映射为统一 HTTP 业务错误。
func mapSearchError(err error) error {
	// 1. 参数错误统一映射为现有参数错误码
	if errors.Is(err, searchdomain.ErrKeywordEmpty) ||
		errors.Is(err, searchdomain.ErrPageInvalid) ||
		errors.Is(err, searchdomain.ErrPageSizeInvalid) {
		return apperrors.ErrParameter
	}

	// 2. 搜索基础设施错误不向外泄漏内部细节
	return apperrors.ErrSearchUnavailable
}
