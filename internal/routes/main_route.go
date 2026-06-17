package routes

import (
	"blog/internal/handler"
	"net/http"
)

// 未来增加新模块，只需在这里加一行，不需要改 InitRoute 的签名
type AppHandler struct {
	UserAuth *handler.UserAuthHandler
	Article  *handler.ArticleHandler
	User     *handler.UserHandler
}

func InitRoute(appHandler *AppHandler) *http.ServeMux {
	mux := http.NewServeMux()

	// 注册用户模块路由
	InitUserRoutes(mux, appHandler.User, appHandler.UserAuth)
	// 注册文章模块路由
	InitArticleRoutes(mux, appHandler.Article)
	return mux
}

type middlewareFunc func(http.Handler) http.Handler

// 作为包内私有工具
// wrap 核心工具：将核心业务 Handler 用多个中间件一层层包起来
// 执行顺序：从右往左（先执行写在后面的中间件，再执行前面的，最后进入核心 Handler）
func wrap(h http.HandlerFunc, middlewares ...middlewareFunc) http.Handler {
	var result http.Handler = h
	// 倒序遍历，确保中间件的执行顺序和传入顺序一致
	for i := len(middlewares) - 1; i >= 0; i-- {
		result = middlewares[i](result)
	}
	return result
}
