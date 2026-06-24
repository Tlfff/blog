package handler

import (
	"blog/internal/repository"
	"blog/internal/service"
	"time"

	"blog/internal/model"
)

func newArticleHandler() *ArticleHandler {
	repo := repository.NewArticleRepository()

	repo.CreateArticle(&model.Article{
		ID:         1,
		Title:      "draft",
		Content:    "draft content",
		AuthorID:   1001,
		Status:     model.Draft,
		AddTime:    time.Now(),
		UpdateTime: time.Now(),
	})

	repo.CreateArticle(&model.Article{
		ID:         2,
		Title:      "published",
		Content:    "published content",
		AuthorID:   1001,
		Status:     model.Published,
		AddTime:    time.Now(),
		UpdateTime: time.Now(),
	})

	return NewArticleHandler(
		service.NewArticleService(repo),
	)
}

// func withUser(req *http.Request) *http.Request {
// 	ctx := auth.SetUserContext(
// 		context.Background(),
// 		&auth.UserContext{
// 			UserID: 1001,
// 			Phone:  "13800138000",
// 			Role:   1,
// 		},
// 	)

// 	return req.WithContext(ctx)
// }

// func TestCreateArticle(t *testing.T) {
// 	h := newArticleHandler()

// 	body := `{
// 		"title":"test",
// 		"content":"hello",
// 		"status":1
// 	}`

// 	req := httptest.NewRequest(
// 		http.MethodPost,
// 		"/article/create",
// 		strings.NewReader(body),
// 	)

// 	req = withUser(req)

// 	w := httptest.NewRecorder()

// 	h.CreateArticle(w, req)

// 	if w.Code != http.StatusOK {
// 		t.Errorf("创建文章失败")
// 	}
// }

// func TestCreateArticleUnauthorized(t *testing.T) {
// 	h := newArticleHandler()

// 	req := httptest.NewRequest(
// 		http.MethodPost,
// 		"/article/create",
// 		strings.NewReader(`{}`),
// 	)

// 	w := httptest.NewRecorder()

// 	h.CreateArticle(w, req)

// 	if w.Code != http.StatusOK {
// 		t.Errorf("请求执行异常")
// 	}
// }

// func TestUpdateArticle(t *testing.T) {
// 	h := newArticleHandler()

// 	body := `{
// 		"id":1,
// 		"title":"new title",
// 		"content":"new content",
// 		"status":1
// 	}`

// 	req := httptest.NewRequest(
// 		http.MethodPost,
// 		"/article/update",
// 		strings.NewReader(body),
// 	)

// 	req = withUser(req)

// 	w := httptest.NewRecorder()

// 	h.UpdateArticle(w, req)

// 	if w.Code != http.StatusOK {
// 		t.Errorf("更新文章失败")
// 	}
// }

// func TestDeleteArticle(t *testing.T) {
// 	h := newArticleHandler()

// 	req := httptest.NewRequest(
// 		http.MethodPost,
// 		"/article/delete",
// 		strings.NewReader(`{"id":1}`),
// 	)

// 	req = withUser(req)

// 	w := httptest.NewRecorder()

// 	h.DeleteArticle(w, req)

// 	if w.Code != http.StatusOK {
// 		t.Errorf("删除文章失败")
// 	}
// }

// func TestPublishArticle(t *testing.T) {
// 	h := newArticleHandler()

// 	req := httptest.NewRequest(
// 		http.MethodPost,
// 		"/article/publish",
// 		strings.NewReader(`{"id":1}`),
// 	)

// 	req = withUser(req)

// 	w := httptest.NewRecorder()

// 	h.PublishArticle(w, req)

// 	if w.Code != http.StatusOK {
// 		t.Errorf("发表文章失败")
// 	}
// }

// func TestGetArticleDetail(t *testing.T) {
// 	h := newArticleHandler()

// 	req := httptest.NewRequest(
// 		http.MethodGet,
// 		"/article/detail?id=2",
// 		nil,
// 	)

// 	w := httptest.NewRecorder()

// 	h.GetArticleDetail(w, req)

// 	if w.Code != http.StatusOK {
// 		t.Errorf("获取文章详情失败")
// 	}
// }

// func TestGetArticleDetailDraft(t *testing.T) {
// 	h := newArticleHandler()

// 	req := httptest.NewRequest(
// 		http.MethodGet,
// 		"/article/detail?id=1",
// 		nil,
// 	)

// 	w := httptest.NewRecorder()

// 	h.GetArticleDetail(w, req)

// 	if w.Code != http.StatusOK {
// 		t.Errorf("请求执行异常")
// 	}
// }

// func TestGetArticleDetailForMe(t *testing.T) {
// 	h := newArticleHandler()

// 	req := httptest.NewRequest(
// 		http.MethodGet,
// 		"/article/me/detail?id=1",
// 		nil,
// 	)

// 	req = withUser(req)

// 	w := httptest.NewRecorder()

// 	h.GetArticleDetailForMe(w, req)

// 	if w.Code != http.StatusOK {
// 		t.Errorf("获取个人文章失败")
// 	}
// }

// func TestGetPublishedList(t *testing.T) {
// 	h := newArticleHandler()

// 	req := httptest.NewRequest(
// 		http.MethodGet,
// 		"/article/list?author_id=1001",
// 		nil,
// 	)

// 	w := httptest.NewRecorder()

// 	h.GetPublishedList(w, req)

// 	if w.Code != http.StatusOK {
// 		t.Errorf("获取发表列表失败")
// 	}
// }

// func TestGetDraftedList(t *testing.T) {
// 	h := newArticleHandler()

// 	req := httptest.NewRequest(
// 		http.MethodGet,
// 		"/article/draft",
// 		nil,
// 	)

// 	req = withUser(req)

// 	w := httptest.NewRecorder()

// 	h.GetDraftedList(w, req)

// 	if w.Code != http.StatusOK {
// 		t.Errorf("获取草稿列表失败")
// 	}
// }

// func TestGetDraftedListUnauthorized(t *testing.T) {
// 	h := newArticleHandler()

// 	req := httptest.NewRequest(
// 		http.MethodGet,
// 		"/article/draft",
// 		nil,
// 	)

// 	w := httptest.NewRecorder()

// 	h.GetDraftedList(w, req)

// 	if w.Code != http.StatusOK {
// 		t.Errorf("请求执行异常")
// 	}
// }
