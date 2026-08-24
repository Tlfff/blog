package routes

import (
	articlehttp "blog/internal/article/interfaces/http"
	commenthttp "blog/internal/comment/interfaces/http"
	likehttp "blog/internal/like/interfaces/http"
	notificationhttp "blog/internal/notification/interfaces/http"
	userhttp "blog/internal/user/interfaces/http"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHTTPRouteContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	app := &AppHandler{
		UserAuth:    userhttp.NewUserAuthHandler(nil),
		Article:     articlehttp.NewArticleHandler(nil, nil),
		User:        userhttp.NewUserHandler(nil, nil),
		Comment:     commenthttp.NewCommentHandler(nil),
		Like:        likehttp.NewLikeHandler(nil, nil),
		Notify:      notificationhttp.NewNotificationHandler(nil),
		ViewHistory: nil,
		Redis:       nil,
	}

	r := gin.New()
	InitRoute(r, app)

	got := make(map[string]struct{})
	for _, route := range r.Routes() {
		got[route.Method+" "+route.Path] = struct{}{}
	}

	want := []string{
		"POST /user/register",
		"POST /user/login",
		"GET /user/profile",
		"GET /article/list",
		"GET /article/hot-rank",
		"GET /comment/list/roots",
		"GET /comment/list/replies",
		"GET /optional/article/detail",
		"GET /auth/my/profile",
		"POST /auth/my/profile/update",
		"POST /auth/my/password/verify",
		"POST /auth/my/password/change",
		"POST /auth/my/account/update",
		"POST /auth/my/avatar/upload-url",
		"POST /auth/my/avatar/confirm",
		"POST /auth/my/logout",
		"POST /auth/comment/create",
		"POST /auth/comment/delete",
		"POST /auth/article/like",
		"POST /auth/article/cancel_like",
		"POST /auth/comment/like",
		"POST /auth/comment/cancel_like",
		"GET /auth/ntf/unread-count",
		"GET /auth/ntf/list",
		"POST /auth/ntf/clear-unread",
		"POST /admin/article/create",
		"POST /admin/article/update",
		"POST /admin/article/delete",
		"POST /admin/article/publish",
		"GET /admin/article/list",
		"GET /admin/article/me/detail",
		"GET /admin/article/trash/list",
		"POST /admin/article/trash/recover",
		"POST /admin/article/trash/clear",
		"POST /admin/article/image/upload-url",
		"POST /admin/comment/delete",
	}

	for _, route := range want {
		if _, ok := got[route]; !ok {
			t.Errorf("缺失路由: %s", route)
		}
	}
	if len(got) != len(want) {
		for route := range got {
			found := false
			for _, expected := range want {
				if route == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("出现未在契约中的路由: %s", route)
			}
		}
	}
}
