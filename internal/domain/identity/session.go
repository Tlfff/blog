package identity

// Session 表示一次登录会话。
type Session struct {
	UserID    uint64
	Role      int8
	LoginTime int64
	IP        string
	Device    string
}
