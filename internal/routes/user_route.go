package routes

import (
	"blog/internal/handler"
	"blog/internal/middleware"
	"net/http"
)

func InitUserRoutes(mux *http.ServeMux, userHandler *handler.UserHandler, userAuthHandler *handler.UserAuthHandler) {

	// ----------------------- 公开接口 -----------------------
	mux.HandleFunc("POST /user/register", userAuthHandler.Register)
	mux.HandleFunc("POST /user/login", userAuthHandler.Login)
	// ----------------------- 用户中心 -----------------------

	mux.Handle(
		"GET /user/profile",
		wrap(userHandler.GetProfile, middleware.AuthMiddleware),
	)

	mux.Handle(
		"POST /user/profile/update",
		wrap(userHandler.UpdateProfile, middleware.AuthMiddleware),
	)

	mux.Handle(
		"POST /user/account/update",
		wrap(userHandler.UpdateAccount, middleware.AuthMiddleware),
	)

	mux.Handle(
		"POST /user/logout",
		wrap(userHandler.Logout, middleware.AuthMiddleware),
	)

}
