package middleware

import (
	"blog/internal/auth"
	"blog/internal/common"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// 防重拦截中间件
// func DuplicateMitigation(expire time.Duration) func(http.Handler) http.Handler {
// 	return func(next http.Handler) http.Handler {
// 		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			// 1. 构件唯一 Key（用户ID + 请求路径）
// 			var key string
// 			if user, ok := auth.GetUserContext(r.Context()); ok {
// 				key = strconv.FormatInt(user.UserID, 10) + "_" + r.URL.Path //转成10进制
// 			}

// 			// 2. 调用防重工具类进行前置拦截
// 			if common.Duplicate.Check(key, expire) {
// 				common.WriteResponse(w, common.CodeDuplicateSubmission, "请勿重复提交请求", nil)
// 				return // 拦截，直接返回，不执行后续的 Handler
// 			}

//				// 3. 放行，进入真正的业务 Handler
//				next.ServeHTTP(w, r)
//			})
//		}
//	}
func DuplicateMitigation(expire time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 从gin上下午中获取用户信息
		user := c.MustGet("currentUser").(*auth.UserContext)
		// 2. 构建唯一的key，将id转为10进制并组装上路由路径
		key := strconv.FormatInt(user.UserID, 10) + "_" + c.FullPath()

		// 3. 调用防重工具类进行前置拦截
		if common.Duplicate.Check(key, expire) {
			c.Error(common.ErrDuplicateSubmission)
			c.Abort()
			return // 拦截，直接返回
		}

		// 4. 放行，进入真正的业务 Handler
		c.Next()
	}
}
