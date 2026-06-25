package routes

import (
	"blog/internal/handler"

	"github.com/gin-gonic/gin"
)

// func InitUserRoutes(r *gin.Engine, userHandler *handler.UserHandler, userAuthHandler *handler.UserAuthHandler) {

// 	// ----------------------- 公开接口 -----------------------
// 	r.POST("/user/register", userAuthHandler.Register)
// 	r.POST("/user/login", userAuthHandler.Login)
// 	// ----------------------- 用户中心 -----------------------

// 	r.GET(
// 		"/user/profile",
// 		middleware.AuthMiddleware(),
// 		userHandler.GetProfile,
// 	)

// 	r.POST(
// 		"/user/profile/update",
// 		middleware.AuthMiddleware(),
// 		userHandler.UpdateProfile,
// 	)

// 	r.POST(
// 		"/user/account/update",
// 		middleware.AuthMiddleware(),
// 		userHandler.UpdateAccount,
// 	)

// 	r.POST(
// 		"/user/logout",
// 		middleware.AuthMiddleware(),
// 		userHandler.Logout,
// 	)

// }

// 用户公开接口
func InitUserPublicRoutes(r *gin.RouterGroup, userAuthHandler *handler.UserAuthHandler, userHandler *handler.UserHandler) {
	r.POST("/user/register", userAuthHandler.Register)
	r.POST("/user/login", userAuthHandler.Login)
	// 查看他人主页
	r.GET("/user/profile", userHandler.GetPublicProfile)
}

// 用户私密接口
func InitUserPrivateRoutes(r *gin.RouterGroup, userHandler *handler.UserHandler) {
	// 查看个人主页
	r.GET("/my/profile", userHandler.GetMyProfile)
	// 更新个人资料
	r.POST("/user/profile/update", userHandler.UpdateProfile)
	// 更新密码
	r.POST("/user/password/update", userHandler.UpdatePassword)
	// 更新账户信息-手机号
	r.POST("/user/account/update", userHandler.UpdateAccount)

	// 退出登录
	r.POST("/user/logout", userHandler.Logout)
}
