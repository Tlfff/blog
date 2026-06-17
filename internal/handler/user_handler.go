package handler

import (
	"blog/internal/auth"
	"blog/internal/common"
	userDto "blog/internal/dto/user"
	"blog/internal/service"
	"encoding/json"
	"net/http"
)

type UserHandler struct {
	user *service.UserService
}

func NewUserHandler(user *service.UserService) *UserHandler {
	return &UserHandler{user: user}
}

// 获取个人资料
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	// 1. 从上下文获取用户信息
	userCtx, ok := auth.GetUserContext(r.Context())
	if !ok {
		common.WriteResponse(w, common.CodeUnauthorized, common.ErrAuthorizationRequired.Error(), nil)
		return
	}
	// 2. 获取用户个人资料
	user, err := h.user.GetProfile(userCtx.UserID)
	if err != nil {
		common.WriteResponse(w, common.GetCodeByError(err), err.Error(), nil)
		return
	}
	common.WriteResponse(w, common.CodeSuccess, "获取成功", userDto.NewUserProfileResponse(user))
}

// 修改基础资料
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	// 1. 解析请求体并放进req
	var req userDto.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteResponse(w, common.CodeBadRequestFormat, common.ErrInvalidRequestBody.Error(), nil)
		return
	}
	// 2. 校验请求参数
	if err := req.Validate(); err != nil {
		common.WriteResponse(w, common.GetCodeByError(err), err.Error(), nil)
		return
	}
	// 3. 从上下文获取用户信息
	userCtx, ok := auth.GetUserContext(r.Context())
	if !ok {
		common.WriteResponse(w, common.CodeUnauthorized, common.ErrAuthorizationRequired.Error(), nil)
		return
	}
	// 4. 更新资料
	err := h.user.UpdateProfile(userCtx.UserID, req.Nickname, req.Avatar)
	if err != nil {
		common.WriteResponse(w, common.GetCodeByError(err), err.Error(), nil)
		return
	}

	common.WriteResponse(w, common.CodeSuccess, "个人资料修改成功", nil)
}

// 修改账户信息（电话、密码）
func (h *UserHandler) UpdateAccount(w http.ResponseWriter, r *http.Request) {
	// 1. 解析请求体并放进req
	var req userDto.UpdateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteResponse(w, common.CodeBadRequestFormat, common.ErrInvalidRequestBody.Error(), nil)
		return
	}
	// 2. 校验请求参数
	if err := req.Validate(); err != nil {
		common.WriteResponse(w, common.GetCodeByError(err), err.Error(), nil)
		return
	}
	// 3. 从上下文获取用户信息
	userCtx, ok := auth.GetUserContext(r.Context())
	if !ok {
		common.WriteResponse(w, common.CodeUnauthorized, common.ErrAuthorizationRequired.Error(), nil)
		return
	}
	// 4. 更新资料
	err := h.user.UpdateAccount(userCtx.UserID, req.Phone, req.OldPassword, req.NewPassword)
	if err != nil {
		common.WriteResponse(w, common.GetCodeByError(err), err.Error(), nil)
		return
	}

	common.WriteResponse(w, common.CodeSuccess, "账号信息修改成功", nil)
}

// 退出登录
// todo :设置黑名单
func (h *UserHandler) Logout(w http.ResponseWriter, r *http.Request) {
	common.WriteResponse(w, common.CodeSuccess, "退出成功", nil)
}
