package http

import (
	commentresult "blog/internal/comment/application/dto"
	"blog/internal/platform/interfaces/http/response"
	"blog/internal/platform/security"
	apperrors "blog/internal/shared/apperrors"
	"context"

	"github.com/gin-gonic/gin"
)

// CommentUsecase 是评论应用用例接口。
type CommentUsecase interface {
	CreateComment(ctx context.Context, articleID, rootID, userID, replyToUserID uint64, content, ip string) (*commentresult.CreateCommentResponse, error)
	ListRootComments(ctx context.Context, articleID, lastID uint64, page, pageSize int, isDesc bool, authorID uint64) (*commentresult.RootCommentListResponse, error)
	ListReplies(ctx context.Context, rootID, lastID uint64, page, pageSize int) (*commentresult.ReplyListResponse, error)
	DeleteComment(ctx context.Context, commentID, userID uint64, isAdmin bool) error
	GetCommentStats(ctx context.Context, commentID uint64) (*commentresult.CommentStatsResponse, error)
}

type CommentHandler struct {
	commentService CommentUsecase
}

func NewCommentHandler(commentService CommentUsecase) *CommentHandler {
	return &CommentHandler{commentService: commentService}
}

// Create 发表评论 (主评论或子评论通用)
func (h *CommentHandler) Create(c *gin.Context) {
	var req CreateCommentRequest
	// 1. 解析并校验请求体r
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.ErrInvalidRequestBody) // 对应 CodeBadRequestFormat[cite: 5, 8]
		return
	}

	// 2. 从上下文中提取当前登录用户信息并转换r
	user := c.MustGet("currentUser").(*auth.UserContext)

	// 3. 将 DTO “拆包”，以完全平铺的参数形式喂给 Service 层
	resp, err := h.commentService.CreateComment(
		c,
		req.ArticleID,
		req.RootID,
		uint64(user.UserID),
		req.ReplyToUserID,
		req.Content,
		c.ClientIP(),
	)
	if err != nil {
		c.Error(err) // 统一错误处理器会处理r
		return
	}

	response.OK(c, "评论成功", resp)
}

// 公开：查看主评论列表 (支持分流：游标与传统跳页)
func (h *CommentHandler) ListRoots(c *gin.Context) {
	var req GetRootListRequest
	// 1. 自动拦截不满足 min=10, max=20 的 page_size 错误r
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(apperrors.ErrInvalidRequestBody) //[cite: 5, 8]
		return
	}

	// 2. 将前端 DTO 翻译映射为 Service 层专属的无标签结构体
	// 3. 获取主评论列表
	res, err := h.commentService.ListRootComments(c, req.ArticleID, req.LastID, req.Page, req.PageSize, req.IsDesc, req.AuthorID)
	if err != nil {
		c.Error(err)
		return
	}

	response.OK(c, "查询成功", res)
}

// ListReplies 公开：展开查看子评论列表 (楼中楼)
func (h *CommentHandler) ListReplies(c *gin.Context) {
	var req GetReplyListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(apperrors.ErrInvalidRequestBody) //[cite: 10]
		return
	}

	res, err := h.commentService.ListReplies(c, req.RootID, req.LastID, req.Page, req.PageSize)
	if err != nil {
		c.Error(err)
		return
	}

	response.OK(c, "查询成功", res)
}

// 用户：删除自己的评论 (软删除)
func (h *CommentHandler) DeleteMyComment(c *gin.Context) {
	var req DeleteCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.ErrInvalidRequestBody) //[cite: 5, 8]
		return
	}

	user := c.MustGet("currentUser").(*auth.UserContext)

	// 转换为平铺参数传参，普通用户校验原作者所有权最后一个参数传 false
	if err := h.commentService.DeleteComment(c, req.ID, uint64(user.UserID), false); err != nil {
		c.Error(err)
		return
	}

	response.OK(c, "评论已成功删除", nil)
}

// 管理员：强制删除违规评论 (软删除)
func (h *CommentHandler) DeleteAdminComment(c *gin.Context) {
	var req AdminDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.ErrInvalidRequestBody) //[cite: 5, 8]
		return
	}

	// 管理员强制覆盖鉴权，最后一个参数传 true，且 userID 传 0 即可
	if err := h.commentService.DeleteComment(c, req.ID, 0, true); err != nil {
		c.Error(err)
		return
	}

	response.OK(c, "管理员已成功处理违规评论", nil)
}
