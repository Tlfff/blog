package middleware

import (
	"blog/internal/auth"
	"blog/internal/common"
	"log"
	"net/http"
	"strings"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. 获取Authorization请求头
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			log.Println("未携带登录凭证，请先登录")
			common.WriteResponse(w, common.CodeUnauthorized, common.ErrAuthorizationRequired.Error(), nil)
			return
		}
		// 2. Bearer 格式校验
		if !strings.HasPrefix(authHeader, "Bearer ") {
			log.Println("Token格式错误,正确格式:Bearer xxx")
			common.WriteResponse(w, common.CodeTokenInvalid, common.ErrTokenInvalid.Error(), nil)
			return
		}

		// 3. 截取Token字符串
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			log.Println("Token不能为空")
			common.WriteResponse(w, common.CodeUnauthorized, common.ErrTokenEmpty.Error(), nil)
			return
		}
		// 4. 校验JWT
		claims, err := auth.VerifyToken(token)
		if err != nil {
			log.Printf("token: %s ,校验失败", token)
			common.WriteResponse(w, common.GetCodeByError(err), err.Error(), nil)
			return
		}
		// 5. 组装用户信息并注入上下文
		userCtx := &auth.UserContext{UserID: claims.UserID, Phone: claims.Phone, Role: claims.Role}
		newCtx := auth.SetUserContext(r.Context(), userCtx)
		// 6. 携带新上下文放行业务处理器
		next.ServeHTTP(w, r.WithContext(newCtx))
	})
}
