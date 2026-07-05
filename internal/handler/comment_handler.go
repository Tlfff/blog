package handler

import (
	"blog/internal/auth"
	"blog/internal/common"
	commentDto "blog/internal/dto/comment"
	"blog/internal/service"

	"github.com/gin-gonic/gin"
)

type CommentHandler struct {
	commentService *service.CommentService
}

func NewCommentHandler(commentService *service.CommentService) *CommentHandler {
	return &CommentHandler{commentService: commentService}
}

// Create 发表评论 (主评论或子评论通用)
// POST /api/v1/comments
func (h *CommentHandler) Create(c *gin.Context) {
	var req commentDto.CreateCommentRequest
	// 1. 解析并校验请求体[cite: 5]
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.ErrInvalidRequestBody) // 对应 CodeBadRequestFormat[cite: 5, 8]
		return
	}

	// 2. 从上下文中提取当前登录用户信息并转换[cite: 5]
	user := c.MustGet("currentUser").(*auth.UserContext) //[cite: 5]

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
		c.Error(err) // 统一错误处理器会处理[cite: 5]
		return
	}

	common.OK(c, "评论成功", resp) //[cite: 5]
}

// ListRoots 公开：查看主评论列表 (支持分流：高性能游标与传统跳页)
// GET /api/v1/comments/roots
func (h *CommentHandler) ListRoots(c *gin.Context) {
	var req commentDto.GetRootListRequest
	// 1. 自动拦截不满足 min=10, max=20 的 page_size 错误[cite: 5]
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(common.ErrInvalidRequestBody) //[cite: 5, 8]
		return
	}

	// 2. 将前端 DTO 翻译映射为 Service 层专属的无标签结构体
	cond := &service.CommentQueryCondition{
		ArticleID: req.ArticleID,
		Page:      req.Page,
		PageSize:  req.PageSize,
		LastID:    req.LastID,
		IsDesc:    req.IsDesc,
		AuthorID:  req.AuthorID,
	}

	// 3. 业务层彻底无视 HTTP 协议
	res, err := h.commentService.GetRootCommentList(c, cond)
	if err != nil {
		c.Error(err) //[cite: 5]
		return
	}

	common.OK(c, "查询成功", res) //[cite: 5]
}

// ListReplies 公开：展开查看子评论列表 (楼中楼)
// GET /api/v1/comments/replies
func (h *CommentHandler) ListReplies(c *gin.Context) {
	var req commentDto.GetReplyListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(common.ErrInvalidRequestBody) //[cite: 10]
		return
	}

	// 转换映射
	cond := &service.ReplyQueryCondition{
		RootID:   req.RootID,
		Page:     req.Page,
		PageSize: req.PageSize,
		LastID:   req.LastID,
	}

	res, err := h.commentService.GetReplyList(c, cond)
	if err != nil {
		c.Error(err) //[cite: 5]
		return
	}

	common.OK(c, "查询成功", res) //[cite: 5]
}

// DeleteMyComment 用户：删除自己的评论 (软删除)
// DELETE /api/v1/comments/my
func (h *CommentHandler) DeleteMyComment(c *gin.Context) {
	var req commentDto.DeleteCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.ErrInvalidRequestBody) //[cite: 5, 8]
		return
	}

	user := c.MustGet("currentUser").(*auth.UserContext) //[cite: 5]

	// 转换为平铺参数传参，普通用户校验原作者所有权最后一个参数传 false
	if err := h.commentService.DeleteComment(c, req.ID, uint64(user.UserID), false); err != nil {
		c.Error(err) //[cite: 5]
		return
	}

	common.OK(c, "评论已成功删除", nil) //[cite: 5]
}

// DeleteAdminComment 管理员：强制删除违规评论 (软删除)
// DELETE /api/v1/comments/admin
func (h *CommentHandler) DeleteAdminComment(c *gin.Context) {
	var req commentDto.AdminDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(common.ErrInvalidRequestBody) //[cite: 5, 8]
		return
	}

	// 管理员强制覆盖鉴权，最后一个参数传 true，且 userID 传 0 即可
	if err := h.commentService.DeleteComment(c, req.ID, 0, true); err != nil {
		c.Error(err) //[cite: 5]
		return
	}

	common.OK(c, "管理员已成功处理违规评论", nil) //[cite: 5]
}
