package user

import (
	"blog/internal/model"
	"blog/pkg/util/ip"
)

// 登录成功响应体
type LoginResponse struct {
	AccessToken string `json:"access_token"`
}

// 返回自己主页信息
type MyProfileResponse struct {
	ID            uint64 `json:"id"`
	Nickname      string `json:"nickname"`        //昵称
	Avatar        string `json:"avatar"`          //头像
	Role          int8   `json:"role"`            //角色
	LastLoginTime int64  `json:"last_login_time"` //最后登录时间
	LastLoginIp   string `json:"last_login_ip"`   //最后登录ip
}

func NewMyProfileResponse(user *model.User) *MyProfileResponse {

	return &MyProfileResponse{
		ID:            user.ID,
		Nickname:      user.Nickname,
		Avatar:        user.Avatar,
		Role:          user.Role,
		LastLoginTime: user.LastLoginTime.Unix(),
		LastLoginIp:   ip.ConvertIPToRegion(user.LastLoginIp),
	}
}

// 返回他人主页信息
type UserProfileResponse struct {
	ID       uint64 `json:"id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

func NewUserProfileResponse(user *model.User) *UserProfileResponse {

	return &UserProfileResponse{
		ID:       user.ID,
		Nickname: user.Nickname,
		Avatar:   user.Avatar,
	}
}

// ----------------------------- 二方服务 --------------------------------------
// 返回用户基本消息
type UserBasicInfoResponse struct {
	ID            uint64 `json:"id"`
	Nickname      string `json:"nickname"`        //昵称
	Avatar        string `json:"avatar"`          //头像
	LastLoginTime int64  `json:"last_login_time"` //最后登录时间
	LastLoginIp   string `json:"last_login_ip"`   //最后登录ip
}

func NewUserBasicInfoResponse(user *model.User) *UserBasicInfoResponse {
	return &UserBasicInfoResponse{
		ID:            user.ID,
		Nickname:      user.Nickname,
		Avatar:        user.Avatar,
		LastLoginTime: user.LastLoginTime.Unix(),
		LastLoginIp:   ip.ConvertIPToRegion(user.LastLoginIp),
	}
}

// 返回用户列表

type UserListItem struct {
	ID       uint64 `json:"id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

type UserListResponse struct {
	List     []*UserListItem `json:"list"`
	Total    uint64          `json:"total"`
	Page     uint64          `json:"page"`      // 页码
	PageSize uint64          `json:"page_size"` // 页面大小
}

// 构造列表响应
func NewUserListResponse(models []*model.User, total, page, page_size uint64) *UserListResponse {
	resp := &UserListResponse{
		List:     make([]*UserListItem, 0),
		Total:    total,
		Page:     page,
		PageSize: page_size,
	}

	for _, m := range models {

		resp.List = append(resp.List, &UserListItem{
			ID:       m.ID,
			Nickname: m.Nickname,
			Avatar:   m.Avatar,
		})
	}
	return resp
}
