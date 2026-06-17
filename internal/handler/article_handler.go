package handler

import (
	"blog/internal/auth"
	"blog/internal/common"
	"blog/internal/dto/article"
	arcticleDto "blog/internal/dto/article"
	"blog/internal/model"
	"blog/internal/service"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

type ArticleHandler struct {
	article *service.ArticleService
}

func NewArticleHandler(article *service.ArticleService) *ArticleHandler {
	return &ArticleHandler{article: article}
}

// 创建文章
func (h *ArticleHandler) CreateArticle(w http.ResponseWriter, r *http.Request) {
	var req arcticleDto.CreateArticleRequest
	// 1. 解析请求体并放进req
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteResponse(w, common.CodeBadRequestFormat, common.ErrInvalidRequestBody.Error(), nil)
		return
	}
	// 2. 校验注册请求参数
	if err := req.Validate(); err != nil {
		common.WriteResponse(w, common.GetCodeByError(err), err.Error(), nil)
		return
	}
	// 3. 从上下文中获取用户信息
	user, ok := auth.GetUserContext(r.Context())
	if !ok {
		common.WriteResponse(w, common.CodeUnauthorized, common.ErrAuthorizationRequired.Error(), nil)
		return
	}

	article := &model.Article{
		Title:      req.Title,
		Content:    req.Content,
		Tags:       req.Tags,
		Status:     req.Status,
		AuthorID:   user.UserID,
		AddTime:    time.Now(),
		UpdateTime: time.Now(),
	}

	if err := h.article.CreateArticle(article); err != nil {
		common.WriteResponse(w, common.CodeInternalServerError, err.Error(), nil)
		return
	}

	common.WriteResponse(w, common.CodeSuccess, "文章创建成功", nil)
}

// 更新文章
func (h *ArticleHandler) UpdateArticle(w http.ResponseWriter, r *http.Request) {
	var req arcticleDto.UpdateArticleRequest
	// 1. 解析请求体并放进req
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteResponse(w, common.CodeBadRequestFormat, common.ErrInvalidRequestBody.Error(), nil)
		return
	}
	// 2. 校验更新请求参数
	if err := req.Validate(); err != nil {
		common.WriteResponse(w, common.GetCodeByError(err), err.Error(), nil)
		return
	}
	// 3. 从上下文中获取用户信息
	user, ok := auth.GetUserContext(r.Context())
	if !ok {
		common.WriteResponse(w, common.CodeUnauthorized, common.ErrAuthorizationRequired.Error(), nil)
		return
	}

	article := &model.Article{
		ID:         req.ID,
		Title:      req.Title,
		Content:    req.Content,
		Tags:       req.Tags,
		Status:     req.Status,
		AuthorID:   user.UserID,
		AddTime:    time.Now(),
		UpdateTime: time.Now(),
	}

	if err := h.article.UpdateArticle(article); err != nil {
		common.WriteResponse(w, common.CodeInternalServerError, err.Error(), nil)
		return
	}

	common.WriteResponse(w, common.CodeSuccess, "文章更新成功", nil)
}

// 删除文章
func (h *ArticleHandler) DeleteArticle(w http.ResponseWriter, r *http.Request) {
	var req arcticleDto.DeleteArticleRequest
	// 1. 解析请求体并放进req
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteResponse(w, common.CodeBadRequestFormat, common.ErrInvalidRequestBody.Error(), nil)
		return
	}
	// 2. 校验删除请求参数
	if err := req.Validate(); err != nil {
		common.WriteResponse(w, common.GetCodeByError(err), err.Error(), nil)
		return
	}
	// 3. 从上下文中获取用户信息
	user, ok := auth.GetUserContext(r.Context())
	if !ok {
		common.WriteResponse(w, common.CodeUnauthorized, common.ErrAuthorizationRequired.Error(), nil)
		return
	}

	if err := h.article.DeleteArticle(req.ID, user.UserID); err != nil {
		common.WriteResponse(w, common.CodeInternalServerError, err.Error(), nil)
		return
	}

	common.WriteResponse(w, common.CodeSuccess, "文章删除成功", nil)
}

// 发表文章
func (h *ArticleHandler) PublishArticle(w http.ResponseWriter, r *http.Request) {
	var req arcticleDto.PublishArticleRequest
	// 1. 解析请求体并放进req
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteResponse(w, common.CodeBadRequestFormat, common.ErrInvalidRequestBody.Error(), nil)
		return
	}
	// 2. 校验发表请求参数
	if err := req.Validate(); err != nil {
		common.WriteResponse(w, common.GetCodeByError(err), err.Error(), nil)
		return
	}
	// 3. 从上下文中获取用户信息
	user, ok := auth.GetUserContext(r.Context())
	if !ok {
		common.WriteResponse(w, common.CodeUnauthorized, common.ErrAuthorizationRequired.Error(), nil)
		return
	}

	if err := h.article.PublishArticle(req.ID, user.UserID); err != nil {
		common.WriteResponse(w, common.CodeInternalServerError, err.Error(), nil)
		return
	}

	common.WriteResponse(w, common.CodeSuccess, "文章发表成功", nil)
}

// 获取文章详情
// 场景A（公开接口）：仅允许查看已发表的文章
func (h *ArticleHandler) GetArticleDetail(w http.ResponseWriter, r *http.Request) {
	// 1. 从 URL Query 解析并构造 DTO
	idStr := r.URL.Query().Get("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	req := article.GetDetailRequest{ID: id}

	// 2. 校验参数
	if err := req.Validate(); err != nil {
		common.WriteResponse(w, common.CodeBadRequestFormat, err.Error(), nil)
		return
	}
	// 3. 获取详情
	article, err := h.article.GetArticle(req.ID)
	if err != nil {
		common.WriteResponse(w, common.CodeInternalServerError, err.Error(), nil)
		return
	}
	// 4. 只能看见已发表的文章
	if article.Status != model.Published {
		common.WriteResponse(w, common.CodeArticleNotFound, common.ErrArticleNotFound.Error(), nil)
		return
	}
	res := arcticleDto.NewArticleDetailResponse(article)
	common.WriteResponse(w, common.CodeSuccess, "查询成功", res)
}

// 场景 B（需要登录）：创作者看自己未发表的文章，如草稿
func (h *ArticleHandler) GetArticleDetailForMe(w http.ResponseWriter, r *http.Request) {
	// 1. 从 URL Query 解析并构造 DTO
	idStr := r.URL.Query().Get("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	req := article.GetDetailRequest{ID: id}
	// 2. 校验参数
	if err := req.Validate(); err != nil {
		common.WriteResponse(w, common.CodeBadRequestFormat, err.Error(), nil)
		return
	}
	// 3. 获取登录用户信息
	userCtx, ok := auth.GetUserContext(r.Context())
	if !ok {
		common.WriteResponse(w, common.CodeUnauthorized, common.ErrAuthorizationRequired.Error(), nil)
		return
	}
	// 4. 获取详情
	articleData, err := h.article.GetArticle(req.ID)
	if err != nil {
		common.WriteResponse(w, common.GetCodeByError(err), err.Error(), nil)
		return
	}

	// 5. 判断自己是否是作者
	if articleData.AuthorID != userCtx.UserID {
		common.WriteResponse(w, common.CodeArticlePermission, common.ErrArticlePermissionDenied.Error(), nil)
		return
	}

	res := article.NewArticleDetailResponse(articleData)
	common.WriteResponse(w, common.CodeSuccess, "查询成功", res)
}

// 获取用户已发表文章列表
func (h *ArticleHandler) GetPublishedList(w http.ResponseWriter, r *http.Request) {
	// 1. 从 URL Query 解析并构造 DTO
	authorIDStr := r.URL.Query().Get("author_id")
	authorID, _ := strconv.ParseInt(authorIDStr, 10, 64)
	req := article.GetPublishListRequest{AuthorID: authorID}

	// 2. 校验参数
	if err := req.Validate(); err != nil {
		common.WriteResponse(w, common.CodeBadRequestFormat, err.Error(), nil)
		return
	}
	articleList, err := h.article.GetPublishedList(req.AuthorID)
	if err != nil {
		common.WriteResponse(w, common.CodeInternalServerError, err.Error(), nil)
		return
	}
	resList := arcticleDto.NewArticleListResponse(articleList)
	common.WriteResponse(w, common.CodeSuccess, "获取发表列表成功", resList)
}

// 获取用户草稿文章列表
func (h *ArticleHandler) GetDraftedList(w http.ResponseWriter, r *http.Request) {
	//  从上下文中获取用户信息
	user, ok := auth.GetUserContext(r.Context())
	if !ok {
		common.WriteResponse(w, common.CodeUnauthorized, common.ErrAuthorizationRequired.Error(), nil)
		return
	}
	articleList, err := h.article.GetDraftedList(user.UserID)
	if err != nil {
		common.WriteResponse(w, common.CodeInternalServerError, err.Error(), nil)
		return
	}

	resList := arcticleDto.NewArticleListResponse(articleList)
	common.WriteResponse(w, common.CodeSuccess, "获取草稿列表成功", resList)
}
