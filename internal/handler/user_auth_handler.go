package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"blog/internal/auth"
	"blog/internal/common"
	"blog/internal/dto/user"
	userDto "blog/internal/dto/user"
	"blog/internal/model"
	"blog/internal/service"
)

type UserAuthHandler struct {
	userAuth *service.UserAuthService
}

func NewUserAuthHandler(userAuth *service.UserAuthService) *UserAuthHandler {
	return &UserAuthHandler{userAuth: userAuth}
}

// Register 处理用户注册请求
func (h *UserAuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req user.RegisterRequest
	// 1. 解析请求体并放进req
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		common.WriteResponse(w, common.CodeBadRequestFormat, common.ErrInvalidRequestBody.Error(), nil)
		return
	}
	// 2. 校验注册请求参数
	if err := req.Validate(); err != nil {
		common.WriteResponse(w, common.GetCodeByError(err), err.Error(), nil)
		return
	}
	// 3. 创建新用户
	user := &model.User{
		ID:         0,
		Nickname:   req.Nickname,
		Phone:      req.Phone,
		Avatar:     "https://example.com/default-avatar.png",
		Role:       int8(model.RoleUser),
		Status:     1,
		AddTime:    time.Now(),
		UpdateTime: time.Now(),
	}
	// 4. 调用服务层进行注册
	err = h.userAuth.Register(user, req.Password)
	if err != nil {
		common.WriteResponse(w, common.GetCodeByError(err), err.Error(), nil)
		return
	}
	common.WriteResponse(w, common.CodeSuccess, "注册成功", nil)
}

// Login 处理用户登录请求
func (h *UserAuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req user.LoginRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		common.WriteResponse(w, common.CodeBadRequestFormat, common.ErrInvalidRequestBody.Error(), nil)
		return
	}
	// 1. 校验登录请求参数
	if err := req.Validate(); err != nil {
		common.WriteResponse(w, common.GetCodeByError(err), err.Error(), nil)
		return
	}
	// 2. 验证用户登录
	user, err := h.userAuth.Login(req.Phone, req.Password)
	if err != nil {
		common.WriteResponse(w, common.GetCodeByError(err), err.Error(), nil)
		return
	}

	// 3. 生成JWT令牌
	token, err := auth.GenerateToken(user.Phone, user.Role, user.ID)
	if err != nil {
		log.Println("生成token失败")
		common.WriteResponse(w, common.CodeInternalServerError, common.ErrSystem.Error(), nil)
		return
	}
	// 4. 封装返回体
	res := userDto.LoginResponse{
		AccessToken: token,
	}
	common.WriteResponse(w, common.CodeSuccess, "登录成功", res)

}

// // Profile 获取当前登录用户信息（测试JWT链路）
// func (h *UserAuthHandler) Profile(w http.ResponseWriter, r *http.Request) {
// 	// 从Context中获取当前登录用户
// 	user, ok := auth.GetUserContext(r.Context())
// 	if !ok {
// 		common.WriteJSON(w, &common.Resoponse{
// 			Code:    common.CodeBadRequest,
// 			Message: "当前未登录,请携带有效Token访问",
// 			Data:    nil,
// 		})
// 		return
// 	}

// 	common.WriteJSON(w, &common.Resoponse{
// 		Code:    common.CodeSuccess,
// 		Message: "获取成功",
// 		Data: map[string]any{
// 			"user_id": user.UserID,
// 			"phone":   user.Phone,
// 			"role":    user.Role,
// 		},
// 	})
// }
