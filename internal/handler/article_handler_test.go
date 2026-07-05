package handler

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
