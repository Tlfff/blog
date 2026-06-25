package user

import (
	"blog/internal/model"
	"time"
)

// 登录成功响应体
type LoginResponse struct {
	AccessToken string `json:"access_token"`
}

// 返回自己主页信息
type MyProfileResponse struct {
	ID            int64     `json:"id"`
	Nickname      string    `json:"nickname"`        //昵称
	Avatar        string    `json:"avatar"`          //头像
	LastLoginTime time.Time `json:"last_login_time"` //最后登录时间
	LastLoginIp   string    `json:"last_login_ip"`   //最后登录ip
}

func NewMyProfileResponse(user *model.User) *MyProfileResponse {

	return &MyProfileResponse{
		ID:            user.ID,
		Nickname:      user.Nickname,
		Avatar:        user.Avatar,
		LastLoginTime: user.LastLoginTime,
		LastLoginIp:   user.LastLoginIp,
	}
}

// 返回他人主页信息
type UserProfileResponse struct {
	ID       int64  `json:"id"`
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
