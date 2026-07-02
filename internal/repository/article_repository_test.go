package repository

// func TestArticleRepository_CreateAndFind(t *testing.T) {
// 	// 1. 核心修复：创建一个临时的纯内存 SQLite 数据库，用来给测试代码发泄数据
// 	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
// 	if err != nil {
// 		t.Fatalf("无法启动内存测试数据库: %v", err)
// 	}

// 	// 2.  自动迁移：让 GORM 默默在内存里把 users 表建出来
// 	_ = db.AutoMigrate(&model.User{})

// 	// 3.  完美对齐升级后的构造函数
// 	repo := NewArticleRepository(db)

// 	// 使用独立 ID: 101
// 	article := &model.Article{
// 		ID:       101,
// 		AuthorID: 100,
// 		Title:    "test",
// 		Status:   model.Draft,
// 	}

// 	err := repo.CreateArticle(article)
// 	if err != nil {
// 		t.Fatalf("创建文章失败: %v", err)
// 	}

// 	res, err := repo.FindArticleByID(101)
// 	if err != nil {
// 		t.Fatalf("查询失败: %v", err)
// 	}
// 	if res == nil {
// 		t.Fatalf("查询结果为空")
// 	}

// 	if res.Title != "test" {
// 		t.Fatalf("文章标题不一致")
// 	}
// }

// func TestArticleRepository_Update(t *testing.T) {
// 	repo := NewArticleRepository()

// 	// 使用独立 ID: 102
// 	article := &model.Article{
// 		ID:       102,
// 		AuthorID: 200,
// 		Title:    "old",
// 		Status:   model.Draft,
// 	}

// 	_ = repo.CreateArticle(article)

// 	article.Title = "new"
// 	err := repo.UpdateArticle(article)
// 	if err != nil {
// 		t.Fatalf("更新失败: %v", err)
// 	}

// 	res, err := repo.FindArticleByID(102)
// 	if err != nil || res == nil {
// 		t.Fatalf("更新后查询失败")
// 	}
// 	if res.Title != "new" {
// 		t.Fatalf("更新未生效")
// 	}
// }

// func TestArticleRepository_Delete(t *testing.T) {
// 	repo := NewArticleRepository()

// 	// 使用独立 ID: 103
// 	article := &model.Article{
// 		ID:       103,
// 		AuthorID: 300,
// 		Title:    "to_be_deleted",
// 		Status:   model.Published,
// 	}

// 	err := repo.CreateArticle(article)
// 	if err != nil {
// 		t.Fatalf("删除前创建文章失败: %v", err)
// 	}

// 	// 执行软删除
// 	err = repo.DeleteArticle(103, 300)
// 	if err != nil {
// 		t.Fatalf("删除失败: %v", err)
// 	}

// 	// 重新捞出来验证状态
// 	res, err := repo.FindArticleByID(103)
// 	if err != nil {
// 		t.Fatalf("删除后查询出错: %v", err)
// 	}
// 	if res == nil {
// 		t.Fatalf("未能成功查询到被软删除的文章主体")
// 	}

// 	if res.Status != model.Deleted {
// 		t.Fatalf("软删除失败, 当前状态为: %v", res.Status)
// 	}
// }

// func TestArticleRepository_GetListByStatus(t *testing.T) {
// 	repo := NewArticleRepository()

// 	// 使用专属当前测试的独立 ID 段：201, 202, 203
// 	// 并且让 AuthorID 统一为 999 避免和上面起冲突
// 	_ = repo.CreateArticle(&model.Article{ID: 201, AuthorID: 999, Status: model.Published})
// 	_ = repo.CreateArticle(&model.Article{ID: 202, AuthorID: 999, Status: model.Draft})
// 	_ = repo.CreateArticle(&model.Article{ID: 203, AuthorID: 888, Status: model.Published}) // 别人发的

// 	// 获取作者 999 的已发布文章列表
// 	list, err := repo.GetListByStatus(999, model.Published)
// 	if err != nil {
// 		t.Fatalf("获取列表失败: %v", err)
// 	}

// 	if len(list) != 1 {
// 		t.Fatalf("列表过滤失败, 预期 1 条，实际拿到 %d 条", len(list))
// 	}
// }
