// Package user 定义用户相关的请求与响应 DTO。
package user

import (
	"blog/internal/platform/ip"
	domain "blog/internal/user/domain"
)

// 登录成功响应体
type LoginResponse struct {
	AccessToken string `json:"access_token"` // 登录成功后颁发的访问令牌
}

// 返回自己主页信息
type MyProfileResponse struct {
	ID            uint64 `json:"id"`              // 用户唯一标识
	Nickname      string `json:"nickname"`        // 用户昵称
	Avatar        string `json:"avatar"`          // 用户头像URL
	Role          int8   `json:"role"`            // 用户角色：1-普通用户 2-管理员
	LastLoginTime int64  `json:"last_login_time"` // 最后登录时间（Unix 秒）
	LastLoginIp   string `json:"last_login_ip"`   // 最后登录IP归属地（已转换为地区文案）
}

// NewMyProfileResponse 构造本人主页响应。
func NewMyProfileResponse(user *domain.User) *MyProfileResponse {

	return &MyProfileResponse{
		ID:            user.ID,
		Nickname:      user.Nickname,
		Avatar:        user.Avatar,
		Role:          user.Role,
		LastLoginTime: user.LastLoginTime.Unix(),
		LastLoginIp:   ip.ConvertIPToRegion(user.LastLoginIP),
	}
}

// 返回他人主页信息
type UserProfileResponse struct {
	ID       uint64 `json:"id"`       // 用户唯一标识
	Nickname string `json:"nickname"` // 用户昵称
	Avatar   string `json:"avatar"`   // 用户头像URL
}

// NewUserProfileResponse 构造他人主页响应。
func NewUserProfileResponse(user *domain.User) *UserProfileResponse {

	return &UserProfileResponse{
		ID:       user.ID,
		Nickname: user.Nickname,
		Avatar:   user.Avatar,
	}
}

// ----------------------------- 二方服务 --------------------------------------
// ----------------------------- 二方服务 --------------------------------------
// 返回用户基本消息
type UserBasicInfoResponse struct {
	ID            uint64 `json:"id"`              // 用户唯一标识
	Nickname      string `json:"nickname"`        // 用户昵称
	Avatar        string `json:"avatar"`          // 用户头像URL
	LastLoginTime int64  `json:"last_login_time"` // 最后登录时间（Unix 秒）
	LastLoginIp   string `json:"last_login_ip"`   // 最后登录IP归属地（已转换为地区文案）
}

// NewUserBasicInfoResponse 构造用户基本信息响应。
func NewUserBasicInfoResponse(user *domain.User) *UserBasicInfoResponse {
	return &UserBasicInfoResponse{
		ID:            user.ID,
		Nickname:      user.Nickname,
		Avatar:        user.Avatar,
		LastLoginTime: user.LastLoginTime.Unix(),
		LastLoginIp:   ip.ConvertIPToRegion(user.LastLoginIP),
	}
}

// UserListItem 是用户列表项响应 DTO。
type UserListItem struct {
	ID       uint64 `json:"id"`       // 用户唯一标识
	Nickname string `json:"nickname"` // 用户昵称
	Avatar   string `json:"avatar"`   // 用户头像URL
}

// UserListResponse 是用户列表响应 DTO。
type UserListResponse struct {
	List     []*UserListItem `json:"list"`      // 用户列表
	Total    uint64          `json:"total"`     // 用户总数
	Page     uint64          `json:"page"`      // 页码
	PageSize uint64          `json:"page_size"` // 页面大小
}

// NewUserListResponse 构造用户列表响应。
func NewUserListResponse(models []*domain.User, total, page, pageSize uint64) *UserListResponse {
	resp := &UserListResponse{
		List:     make([]*UserListItem, 0),
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}

	for _, m := range models {

		// 2. 仅映射对外公开字段
		resp.List = append(resp.List, &UserListItem{
			ID:       m.ID,
			Nickname: m.Nickname,
			Avatar:   m.Avatar,
		})
	}
	return resp
}
