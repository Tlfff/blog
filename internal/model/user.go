package model

import "time"

type User struct {
	ID         int64     `json:"id"`          // 用户唯一标识
	Nickname   string    `json:"nickname"`    // 用户昵称
	Phone      string    `json:"phone"`       // 手机号（登录账号）
	Password   string    `json:"-"`           // PBKDF2加密后的密码：算法$迭代次数$Salt$Hash
	Avatar     string    `json:"avatar"`      // 用户头像URL
	Role       int8      `json:"role"`        // 用户角色：1-普通用户 2-管理员
	AddTime    time.Time `json:"add_time"`    // 注册时间
	UpdateTime time.Time `json:"update_time"` // 信息最后修改时间
	Status     int8      `json:"status"`      // 用户状态：0-删除/禁用 1-正常
}
