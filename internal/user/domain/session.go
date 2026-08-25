package domain

// Session 表示一次登录会话（值对象）。
type Session struct {
	UserID    uint64 // 会话所属用户ID
	Role      int8   // 登录时的用户角色：1-普通用户 2-管理员
	LoginTime int64  // 登录时间（Unix 秒）
	IP        string // 登录来源IP
	Device    string // 登录设备标识
}
