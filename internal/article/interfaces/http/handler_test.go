package http

import (
	articleapp "blog/internal/article/app"
	articledto "blog/internal/article/app/dto"
	"blog/internal/platform/security"
	apperrors "blog/internal/shared/apperrors"
	"bytes"
	"context"
	"errors"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// fakeArticleUsecase 是 Article HTTP Handler 测试使用的应用用例。
type fakeArticleUsecase struct {
	ArticleUsecase                                              // 未使用方法通过嵌入接口保留
	initializeResult *articledto.InitializeArticleResponse      // 初始化文章返回结果
	initializeErr    error                                      // 初始化文章返回错误
	uploadResult     *articledto.ImageUploadCredentialsResponse // 批量凭证返回结果
	uploadErr        error                                      // 批量凭证返回错误
	receivedCommand  articleapp.GetImageUploadURLsCommand       // Handler 传入的批量凭证命令
}

// InitializeArticle 返回测试初始化结果。
func (f *fakeArticleUsecase) InitializeArticle(context.Context, uint64) (*articledto.InitializeArticleResponse, error) {
	// 1. 返回预设初始化结果
	return f.initializeResult, f.initializeErr
}

// GetImageUploadURLs 记录批量凭证命令并返回测试结果。
func (f *fakeArticleUsecase) GetImageUploadURLs(_ context.Context, command articleapp.GetImageUploadURLsCommand) (*articledto.ImageUploadCredentialsResponse, error) {
	// 1. 记录命令并返回预设批量凭证结果
	f.receivedCommand = command
	return f.uploadResult, f.uploadErr
}

// TestArticleHandlerInitializeArticle 验证初始化文章响应映射。
func TestArticleHandlerInitializeArticle(t *testing.T) {
	// 1. 准备初始化成功的 Handler 测试上下文
	gin.SetMode(gin.TestMode)
	usecase := &fakeArticleUsecase{initializeResult: &articledto.InitializeArticleResponse{ArticleID: 7}}
	handler := NewArticleHandler(usecase, nil)
	ctx, recorder := newArticleHandlerContext(nethttp.MethodPost, "/article/init", "")
	ctx.Set("currentUser", &auth.UserContext{UserID: 100})

	// 2. 执行请求并校验文章 ID 响应
	handler.InitializeArticle(ctx)

	if recorder.Code != nethttp.StatusOK || !strings.Contains(recorder.Body.String(), `"article_id":7`) {
		t.Fatalf("初始化文章响应错误: code=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

// TestArticleHandlerGetImageUploadURLs 验证批量凭证请求和响应字段映射。
func TestArticleHandlerGetImageUploadURLs(t *testing.T) {
	// 1. 准备批量凭证成功响应和请求上下文
	gin.SetMode(gin.TestMode)
	usecase := &fakeArticleUsecase{uploadResult: &articledto.ImageUploadCredentialsResponse{
		Files: []articledto.ImageUploadCredential{{
			ClientID:  "image-1",
			UploadURL: "https://upload.example/article/7/a.png",
			URL:       "https://cdn.example/article/7/a.png",
		}},
	}}
	handler := NewArticleHandler(usecase, nil)
	body := `{"article_id":7,"files":[{"client_id":"image-1","file_ext":"png"}]}`
	ctx, recorder := newArticleHandlerContext(nethttp.MethodPost, "/article/image/upload-urls", body)
	ctx.Set("currentUser", &auth.UserContext{UserID: 100})

	// 2. 执行请求并校验命令及响应字段映射
	handler.GetImageUploadURLs(ctx)

	if len(ctx.Errors) != 0 {
		t.Fatalf("批量凭证请求不应返回错误: %v", ctx.Errors)
	}
	if usecase.receivedCommand.ArticleID != 7 || usecase.receivedCommand.AuthorID != 100 || len(usecase.receivedCommand.Files) != 1 {
		t.Fatalf("批量凭证命令映射错误: %+v", usecase.receivedCommand)
	}
	if !strings.Contains(recorder.Body.String(), `"client_id":"image-1"`) || !strings.Contains(recorder.Body.String(), `"upload_url":`) {
		t.Fatalf("批量凭证响应字段错误: %s", recorder.Body.String())
	}
}

// TestArticleHandlerGetImageUploadURLsErrors 验证请求校验和应用错误传递。
func TestArticleHandlerGetImageUploadURLsErrors(t *testing.T) {
	// 1. 验证空文件列表被 HTTP 参数校验拒绝
	gin.SetMode(gin.TestMode)
	invalidUsecase := &fakeArticleUsecase{}
	invalidHandler := NewArticleHandler(invalidUsecase, nil)
	invalidCtx, _ := newArticleHandlerContext(nethttp.MethodPost, "/article/image/upload-urls", `{"article_id":7,"files":[]}`)
	invalidHandler.GetImageUploadURLs(invalidCtx)
	if len(invalidCtx.Errors) == 0 || !errors.Is(invalidCtx.Errors.Last().Err, apperrors.ErrInvalidRequestBody) {
		t.Fatalf("空文件列表错误不正确: %v", invalidCtx.Errors)
	}

	// 2. 验证 Application 权限错误由 Handler 原样传递
	permissionUsecase := &fakeArticleUsecase{uploadErr: apperrors.ErrArticlePermissionDenied}
	permissionHandler := NewArticleHandler(permissionUsecase, nil)
	permissionCtx, _ := newArticleHandlerContext(nethttp.MethodPost, "/article/image/upload-urls", `{"article_id":7,"files":[{"client_id":"image-1","file_ext":"png"}]}`)
	permissionCtx.Set("currentUser", &auth.UserContext{UserID: 200})
	permissionHandler.GetImageUploadURLs(permissionCtx)
	if len(permissionCtx.Errors) == 0 || !errors.Is(permissionCtx.Errors.Last().Err, apperrors.ErrArticlePermissionDenied) {
		t.Fatalf("权限错误未传递: %v", permissionCtx.Errors)
	}
}

// newArticleHandlerContext 创建 Article Handler 测试上下文。
func newArticleHandlerContext(method, target, body string) (*gin.Context, *httptest.ResponseRecorder) {
	// 1. 创建请求和响应记录器
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")

	// 2. 组装 Gin 测试上下文
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request
	return ctx, recorder
}

// func TestArticleHandler_AllRoutes(t *testing.T) {
// 	// 1. 核心修复：创建一个临时的纯内存 SQLite 数据库，用来给测试代码发泄数据
// 	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
// 	if err != nil {
// 		t.Fatalf("无法启动内存测试数据库: %v", err)
// 	}

// 	// 2.  自动迁移：让 GORM 默默在内存里把 users 表建出来
// 	_ = db.AutoMigrate(&model.User{})

// 	// 3.  完美对齐升级后的构造函数
// 	articleRepo := repository.NewArticleRepository(db)
// 	articleService := service.NewArticleService(articleRepo)
// 	h := NewArticleHandler(articleService)

// 	// 4. 🎯 大表格：覆盖增、删、改、查、发布、垃圾箱等所有 if 分支
// 	tests := []struct {
// 		name           string
// 		run            func(c *gin.Context)
// 		method         string
// 		path           string
// 		body           interface{}
// 		ctxUser        *auth.UserContext // 模拟当前登录用户
// 		expectContains string
// 	}{
// 		// ==================== 📝 1. 创建文章 (CreateArticle) ====================
// 		{
// 			name:           "1. 创建文章-JSON参数解析错误(触发第一个if)",
// 			run:            h.CreateArticle,
// 			method:         "POST",
// 			path:           "/article/create",
// 			body:           "坏的JSON字符串",
// 			ctxUser:        &auth.UserContext{UserID: 1},
// 			expectContains: "",
// 		},
// 		{
// 			name:   "2. 创建文章-成功通关",
// 			run:    h.CreateArticle,
// 			method: "POST",
// 			path:   "/article/create",
// 			body: article.CreateArticleRequest{
// 				Title:   "Go单测指南",
// 				Content: "今天林风努力把覆盖率刷到了100%...",
// 				Tags:    []string{"Go,Test"},
// 				Status:  int8(model.Draft), // 初始为草稿状态
// 			},
// 			ctxUser:        &auth.UserContext{UserID: 1},
// 			expectContains: `"文章创建成功"`,
// 		},

// 		// ==================== 🔄 2. 更新文章 (UpdateArticle) ====================
// 		{
// 			name:   "3. 更新文章-成功通关",
// 			run:    h.UpdateArticle,
// 			method: "PUT",
// 			path:   "/article/update",
// 			body: article.UpdateArticleRequest{
// 				ID:      1, // 刚才创建的第一篇文章
// 				Title:   "Go单测指南(已修改)",
// 				Content: "修改后的内容...",
// 				Status:  int8(model.Draft),
// 			},
// 			ctxUser:        &auth.UserContext{UserID: 1},
// 			expectContains: `"文章更新成功"`,
// 		},

// 		// ==================== 📢 3. 发布文章 (PublishArticle) ====================
// 		{
// 			name:   "4. 发布文章-成功通关",
// 			run:    h.PublishArticle,
// 			method: "POST",
// 			path:   "/article/publish",
// 			body: article.PublishArticleRequest{
// 				ID: 1,
// 			},
// 			ctxUser:        &auth.UserContext{UserID: 1},
// 			expectContains: `"文章发表成功"`,
// 		},

// 		// ==================== 🔍 4. 获取文章详情 (GetArticleDetail) ====================
// 		{
// 			name:           "5. 公开查看详情-ID无效Query绑定错误(第一个if)",
// 			run:            h.GetArticleDetail,
// 			method:         "GET",
// 			path:           "/article/detail?id=abc", // 传入非法 id 字符串
// 			ctxUser:        nil,
// 			expectContains: "",
// 		},
// 		{
// 			name:           "6. 公开查看详情-成功通关(已发布状态)",
// 			run:            h.GetArticleDetail,
// 			method:         "GET",
// 			path:           "/article/detail?id=1",
// 			ctxUser:        nil,
// 			expectContains: `"查询成功"`,
// 		},

// 		// ==================== 🛡️ 5. 管理员查看详情 (GetArticleDetailForMe) ====================
// 		{
// 			name:           "7. 管理员查看详情-非作者查看无权限(判断AuthorID != UserID)",
// 			run:            h.GetArticleDetailForMe,
// 			method:         "GET",
// 			path:           "/article/detail/me?id=1",
// 			ctxUser:        &auth.UserContext{UserID: 999}, // 故意换成 999 号非作者用户
// 			expectContains: "",
// 		},
// 		{
// 			name:           "8. 管理员查看详情-作者本人查看成功",
// 			run:            h.GetArticleDetailForMe,
// 			method:         "GET",
// 			path:           "/article/detail/me?id=1",
// 			ctxUser:        &auth.UserContext{UserID: 1}, // 1 号作者本人
// 			expectContains: `"查询成功"`,
// 		},

// 		// ==================== 📊 6. 获取文章列表相关接口 ====================
// 		{
// 			name:           "9. 获取用户已发表文章列表-成功通关",
// 			run:            h.GetPublishedList,
// 			method:         "GET",
// 			path:           "/article/list/published?author_id=1",
// 			ctxUser:        nil,
// 			expectContains: `"获取发表列表成功"`,
// 		},
// 		{
// 			name:           "10. 管理者获取文章列表-成功通关",
// 			run:            h.GetAdminList,
// 			method:         "GET",
// 			path:           "/article/list/admin?status=1",
// 			ctxUser:        &auth.UserContext{UserID: 1},
// 			expectContains: `"获取文章列表成功"`,
// 		},

// 		// ==================== 🗑️ 7. 垃圾箱全生命周期 (Delete/Trash/Recover/Clear) ====================
// 		{
// 			name:   "11. 软删除文章移入垃圾箱-成功",
// 			run:    h.DeleteArticle,
// 			method: "DELETE",
// 			path:   "/article/delete",
// 			body: article.DeleteArticleRequest{
// 				ID: 1,
// 			},
// 			ctxUser:        &auth.UserContext{UserID: 1},
// 			expectContains: `"文章删除成功"`,
// 		},
// 		{
// 			name:           "12. 查看垃圾箱列表-成功(固定读取状态为Deleted)",
// 			run:            h.GetTrashList,
// 			method:         "GET",
// 			path:           "/article/trash",
// 			ctxUser:        &auth.UserContext{UserID: 1},
// 			expectContains: `"获取垃圾箱列表成功"`,
// 		},
// 		{
// 			name:           "13. 恢复垃圾箱中的文章-成功",
// 			run:            h.RecoverArticle,
// 			method:         "POST",
// 			path:           "/article/recover?id=1",
// 			ctxUser:        &auth.UserContext{UserID: 1},
// 			expectContains: `"恢复文章成功"`,
// 		},
// 		{
// 			name:           "14. 再次软删除以备硬删除",
// 			run:            h.DeleteArticle,
// 			method:         "DELETE",
// 			path:           "/article/delete",
// 			body:           article.DeleteArticleRequest{ID: 1},
// 			ctxUser:        &auth.UserContext{UserID: 1},
// 			expectContains: `"文章删除成功"`,
// 		},
// 		{
// 			name:           "15. 硬删除彻底清除文章-成功",
// 			run:            h.ClearArticle,
// 			method:         "DELETE",
// 			path:           "/article/clear?id=1",
// 			ctxUser:        &auth.UserContext{UserID: 1},
// 			expectContains: `"删除文章成功"`,
// 		},
// 	}

// 	// 3. 🤖 自动化驱动引擎
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			// 复用在 user_test.go 里写的全局 makeTestContext 脚手架
// 			c, w := makeTestContext(tt.method, tt.path, tt.body, tt.ctxUser)

// 			// 轰炸对应的路由方法
// 			tt.run(c)

// 			// 异常拦截日志捕获
// 			actualBody := w.Body.String()
// 			if actualBody == "" && len(c.Errors) > 0 {
// 				actualBody = "[被 c.Error 拦截] 原因: " + c.Errors.Last().Error()
// 			}

// 			// 结果校验断言
// 			if tt.expectContains != "" && !bytes.Contains(w.Body.Bytes(), []byte(tt.expectContains)) {
// 				t.Errorf("用例 [%s] 失败!\n预期包含: %s\n实际返回: %s", tt.name, tt.expectContains, actualBody)
// 			}
// 		})
// 	}
// }
