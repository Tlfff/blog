package auth

import "github.com/golang-jwt/jwt/v5"

// 存储JWT的Payload部分的结构体
type Claims struct {
	UserID uint64 `json:"user_id"`
	Phone  string `json:"phone"`
	Role   int8   `json:"role"`

	// 内嵌标准Claims，提供iss(签发者)、iat(签发时间)、exp(过期时间)等字段，校验逻辑由库统一完成
	jwt.RegisteredClaims
}
