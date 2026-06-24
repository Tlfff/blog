package handler

import (
	"blog/internal/repository"
	"blog/internal/service"
)

func newUserAuthHandler() *UserAuthHandler {
	repo := repository.NewUserRepository()

	return NewUserAuthHandler(
		service.NewUserAuthService(repo),
	)
}

// func TestRegister(t *testing.T) {
// 	h := newUserAuthHandler()

// 	body := `{
// 		"nickname":"test",
// 		"phone":"13800138000",
// 		"password":"123456",
// 		"role":1
// 	}`

// 	req := httptest.NewRequest(
// 		http.MethodPost,
// 		"/user/register",
// 		strings.NewReader(body),
// 	)

// 	w := httptest.NewRecorder()

// 	h.Register(w, req)

// 	if w.Code != http.StatusOK {
// 		t.Errorf("注册失败")
// 	}
// }
// func TestRegisterInvalidRequest(t *testing.T) {
// 	h := newUserAuthHandler()

// 	req := httptest.NewRequest(
// 		http.MethodPost,
// 		"/user/register",
// 		strings.NewReader(`{}`),
// 	)

// 	w := httptest.NewRecorder()

// 	h.Register(w, req)

// 	if w.Code != http.StatusOK {
// 		t.Errorf("请求执行异常")
// 	}
// }
// func TestRegisterBadJSON(t *testing.T) {
// 	h := newUserAuthHandler()

// 	req := httptest.NewRequest(
// 		http.MethodPost,
// 		"/user/register",
// 		strings.NewReader(`{nickname}`),
// 	)

// 	w := httptest.NewRecorder()

// 	h.Register(w, req)

// 	if w.Code != http.StatusOK {
// 		t.Errorf("请求执行异常")
// 	}
// }
// func TestRegisterDuplicateUser(t *testing.T) {
// 	h := newUserAuthHandler()

// 	body := `{
// 		"nickname":"test",
// 		"phone":"13800138001",
// 		"password":"123456",
// 		"role":1
// 	}`

// 	req1 := httptest.NewRequest(
// 		http.MethodPost,
// 		"/user/register",
// 		strings.NewReader(body),
// 	)

// 	w1 := httptest.NewRecorder()

// 	h.Register(w1, req1)

// 	req2 := httptest.NewRequest(
// 		http.MethodPost,
// 		"/user/register",
// 		strings.NewReader(body),
// 	)

// 	w2 := httptest.NewRecorder()

// 	h.Register(w2, req2)

// 	if w2.Code != http.StatusOK {
// 		t.Errorf("请求执行异常")
// 	}
// }
// func TestLogin(t *testing.T) {
// 	h := newUserAuthHandler()

// 	registerBody := `{
// 		"nickname":"login",
// 		"phone":"13800138002",
// 		"password":"123456",
// 		"role":1
// 	}`

// 	reqRegister := httptest.NewRequest(
// 		http.MethodPost,
// 		"/user/register",
// 		strings.NewReader(registerBody),
// 	)

// 	wRegister := httptest.NewRecorder()

// 	h.Register(wRegister, reqRegister)

// 	loginBody := `{
// 		"phone":"13800138002",
// 		"password":"123456"
// 	}`

// 	req := httptest.NewRequest(
// 		http.MethodPost,
// 		"/user/login",
// 		strings.NewReader(loginBody),
// 	)

// 	w := httptest.NewRecorder()

// 	h.Login(w, req)

// 	if w.Code != http.StatusOK {
// 		t.Errorf("登录失败")
// 	}
// }
// func TestLoginWrongPassword(t *testing.T) {
// 	h := newUserAuthHandler()

// 	registerBody := `{
// 		"nickname":"login",
// 		"phone":"13800138003",
// 		"password":"123456",
// 		"role":1
// 	}`

// 	reqRegister := httptest.NewRequest(
// 		http.MethodPost,
// 		"/user/register",
// 		strings.NewReader(registerBody),
// 	)

// 	wRegister := httptest.NewRecorder()

// 	h.Register(wRegister, reqRegister)

// 	loginBody := `{
// 		"phone":"13800138003",
// 		"password":"654321"
// 	}`

// 	req := httptest.NewRequest(
// 		http.MethodPost,
// 		"/user/login",
// 		strings.NewReader(loginBody),
// 	)

// 	w := httptest.NewRecorder()

// 	h.Login(w, req)

// 	if w.Code != http.StatusOK {
// 		t.Errorf("请求执行异常")
// 	}
// }
// func TestLoginUserNotFound(t *testing.T) {
// 	h := newUserAuthHandler()

// 	loginBody := `{
// 		"phone":"19999999999",
// 		"password":"123456"
// 	}`

// 	req := httptest.NewRequest(
// 		http.MethodPost,
// 		"/user/login",
// 		strings.NewReader(loginBody),
// 	)

// 	w := httptest.NewRecorder()

// 	h.Login(w, req)

// 	if w.Code != http.StatusOK {
// 		t.Errorf("请求执行异常")
// 	}
// }
