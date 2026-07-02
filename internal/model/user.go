package model

import "time"

type User struct {
	ID            uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`           // 用户唯一标识
	Nickname      string    `gorm:"column:nickname" json:"nickname"`                        // 用户昵称
	Phone         string    `gorm:"column:phone" json:"phone"`                              // 手机号
	Password      string    `gorm:"column:password" json:"-"`                               // PBKDF2加密后的密码：算法$迭代次数$Salt$Hash
	Avatar        string    `gorm:"column:avatar" json:"avatar"`                            // 用户头像URL
	Role          int8      `gorm:"column:role" json:"role"`                                // 用户角色：1-普通用户 2-管理员
	Status        int8      `gorm:"column:status" json:"status"`                            // 用户状态：0-删除/禁用 1-正常
	LastLoginIp   string    `gorm:"column:last_login_ip" json:"last_login_ip"`              // 上一次最后登录的IP地址
	LastLoginTime time.Time `gorm:"column:last_login_time" json:"last_login_time"`          // 上一次最后登录的时间
	CreatedTime   time.Time `gorm:"column:created_time;autoCreateTime" json:"created_time"` // 创建时间
	UpdatedTime   time.Time `gorm:"column:updated_time;autoUpdateTime" json:"updated_time"` // 更新时间
}
