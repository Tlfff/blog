package user

import (
	"blog/internal/model"
	"time"
)

// 登录成功响应体
type LoginResponse struct {
	AccessToken string `json:"access_token"`
}

type UserProfileResponse struct {
	ID         int64     `json:"id"`
	Nickname   string    `json:"nickname"`
	Phone      string    `json:"phone"`
	Avatar     string    `json:"avatar"`
	Role       int8      `json:"role"`
	Status     int8      `json:"status"`
	AddTime    time.Time `json:"add_time"`
	UpdateTime time.Time `json:"update_time"`
}

func NewUserProfileResponse(user *model.User) *UserProfileResponse {

	return &UserProfileResponse{
		ID:         user.ID,
		Nickname:   user.Nickname,
		Phone:      user.Phone,
		Avatar:     user.Avatar,
		Role:       user.Role,
		Status:     user.Status,
		AddTime:    user.AddTime,
		UpdateTime: user.UpdateTime,
	}
}
