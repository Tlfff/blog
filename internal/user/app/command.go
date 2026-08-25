package app

// RegisterCommand 表示用户注册用例输入。
type RegisterCommand struct {
	Phone    string // 手机号
	Password string // 明文密码
	Nickname string // 用户昵称
	ClientIP string // 注册来源 IP
}

// LoginCommand 表示用户登录用例输入。
type LoginCommand struct {
	Phone      string // 手机号，可以为空
	Nickname   string // 用户昵称，可以为空
	Password   string // 明文密码
	ClientIP   string // 登录来源 IP
	Device     string // 登录设备标识
	RememberMe bool   // 是否延长登录会话有效期
}

// UpdateProfileCommand 表示修改用户公开资料用例输入。
type UpdateProfileCommand struct {
	UserID   uint64 // 当前用户唯一标识
	Nickname string // 新昵称
	Avatar   string // 新头像地址
}

// ChangePasswordCommand 表示使用一次性凭证修改密码的用例输入。
type ChangePasswordCommand struct {
	UserID       uint64 // 当前用户唯一标识
	ChangeToken  string // 一次性密码修改凭证
	NewPassword  string // 新明文密码
	CurrentToken string // 当前登录 Token
}
