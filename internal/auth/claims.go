package auth

// 存储JWT的Payload部分的结构体
type Claims struct {
	UserID uint64 `json:"user_id"`
	Phone  string `json:"phone"`
	Role   int8   `json:"role"`

	Iss string `json:"iss"` //JWT的签发者
	Iat int64  `json:"iat"` //JWT的签发时间，单位为秒
	Exp int64  `json:"exp"` //JWT的过期时间，单位为秒
}
