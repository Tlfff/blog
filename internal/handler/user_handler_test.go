package handler

import (
	"blog/internal/auth"
	"blog/internal/model"
	"blog/internal/repository"
	"blog/internal/service"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newUserHandler() *UserHandler {
	userRepo := repository.NewUserRepository()
	userRepo.CreateUser(&model.User{
		ID:       1,
		Nickname: "测试用户",
		Phone:    "13800138000",
		Password: "123456",
		Status:   1,
	})

	return NewUserHandler(
		service.NewUserService(userRepo),
	)
}

func withUserContext(req *http.Request) *http.Request {
	ctx := auth.SetUserContext(
		context.Background(),
		&auth.UserContext{
			UserID: 1,
			Phone:  "13800138000",
			Role:   1,
		},
	)

	return req.WithContext(ctx)
}
func TestGetProfile(t *testing.T) {
	h := newUserHandler()

	req := httptest.NewRequest(
		http.MethodGet,
		"/user/profile",
		nil,
	)

	req = withUserContext(req)

	w := httptest.NewRecorder()

	h.GetProfile(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("获取个人资料失败")
	}
}

func TestGetProfileUnauthorized(t *testing.T) {
	h := newUserHandler()

	req := httptest.NewRequest(
		http.MethodGet,
		"/user/profile",
		nil,
	)

	w := httptest.NewRecorder()

	h.GetProfile(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("请求执行异常")
	}
}
func TestUpdateProfile(t *testing.T) {
	h := newUserHandler()

	body := `{
		"nickname":"新昵称",
		"avatar":"avatar.png"
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/user/profile/update",
		strings.NewReader(body),
	)

	req = withUserContext(req)

	w := httptest.NewRecorder()

	h.UpdateProfile(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("修改资料失败")
	}
}

func TestUpdateProfileInvalidRequest(t *testing.T) {
	h := newUserHandler()

	req := httptest.NewRequest(
		http.MethodPost,
		"/user/profile/update",
		strings.NewReader(`{}`),
	)

	req = withUserContext(req)

	w := httptest.NewRecorder()

	h.UpdateProfile(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("请求执行异常")
	}
}

func TestUpdateProfileUnauthorized(t *testing.T) {
	h := newUserHandler()

	req := httptest.NewRequest(
		http.MethodPost,
		"/user/profile/update",
		strings.NewReader(`{}`),
	)

	w := httptest.NewRecorder()

	h.UpdateProfile(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("请求执行异常")
	}
}
func TestUpdateAccount(t *testing.T) {
	h := newUserHandler()

	body := `{
		"phone":"13900000000",
		"old_password":"123456",
		"new_password":"654321"
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/user/account/update",
		strings.NewReader(body),
	)

	req = withUserContext(req)

	w := httptest.NewRecorder()

	h.UpdateAccount(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("修改账号失败")
	}
}
func TestUpdateAccountWrongPassword(t *testing.T) {
	h := newUserHandler()

	body := `{
		"phone":"13900000000",
		"old_password":"wrong",
		"new_password":"654321"
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/user/account/update",
		strings.NewReader(body),
	)

	req = withUserContext(req)

	w := httptest.NewRecorder()

	h.UpdateAccount(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("请求执行异常")
	}
}
func TestUpdateAccountInvalidRequest(t *testing.T) {
	h := newUserHandler()

	req := httptest.NewRequest(
		http.MethodPost,
		"/user/account/update",
		strings.NewReader(`{}`),
	)

	req = withUserContext(req)

	w := httptest.NewRecorder()

	h.UpdateAccount(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("请求执行异常")
	}
}
func TestUpdateAccountUnauthorized(t *testing.T) {
	h := newUserHandler()

	req := httptest.NewRequest(
		http.MethodPost,
		"/user/account/update",
		nil,
	)

	w := httptest.NewRecorder()

	h.UpdateAccount(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("请求执行异常")
	}
}
func TestLogout(t *testing.T) {
	h := newUserHandler()

	req := httptest.NewRequest(
		http.MethodPost,
		"/user/logout",
		nil,
	)

	w := httptest.NewRecorder()

	h.Logout(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("退出登录失败")
	}
}
