package service

import (
	"blog/internal/common"
	"blog/internal/model"
	"blog/internal/repository"
	"testing"
)

// 1. 测试创建文章
func TestArticleService_CreateArticle(t *testing.T) {
	repo := repository.NewArticleRepository()
	service := NewArticleService(repo)

	art := &model.Article{
		ID:       1,
		AuthorID: 100,
		Title:    "Go 单测指南",
		Content:  "单元测试非常重要",
		Status:   model.Draft,
	}

	err := service.CreateArticle(art)
	if err != nil {
		t.Fatalf("预期创建成功，但收到错误: %v", err)
	}

	// 从底层 repo 取出验证落盘和时间戳注入
	saved, err := repo.FindArticleByID(1)
	if err != nil || saved == nil {
		t.Fatalf("持久化验证失败，未找到文章")
	}
	if saved.AddTime.IsZero() || saved.UpdateTime.IsZero() {
		t.Errorf("系统未能成功自动注入业务时间戳")
	}
}

// 2. 测试更新文章（包含纵深权限校验与状态熔断）
func TestArticleService_UpdateArticle(t *testing.T) {
	repo := repository.NewArticleRepository()
	service := NewArticleService(repo)

	t.Run("成功更新内容", func(t *testing.T) {
		// 预埋一条草稿
		_ = repo.CreateArticle(&model.Article{
			ID:       2,
			AuthorID: 100,
			Title:    "老标题",
			Status:   model.Draft,
		})

		updateData := &model.Article{
			ID:       2,
			AuthorID: 100, // 作者一致
			Title:    "新标题",
			Status:   model.Draft,
		}

		if err := service.UpdateArticle(updateData); err != nil {
			t.Fatalf("更新失败: %v", err)
		}
	})

	t.Run("拦截越权篡改", func(t *testing.T) {
		_ = repo.CreateArticle(&model.Article{
			ID:       3,
			AuthorID: 100, // 原作者是 100
			Title:    "机密文章",
			Status:   model.Draft,
		})

		hackerData := &model.Article{
			ID:       3,
			AuthorID: 999, // 恶意第三方
			Title:    "试图篡改标题",
		}

		err := service.UpdateArticle(hackerData)
		if err != common.ErrArticlePermissionDenied {
			t.Errorf("预期返回越权错误 %v，实际得到: %v", common.ErrArticlePermissionDenied, err)
		}
	})

	t.Run("锁定已被删除的文章", func(t *testing.T) {
		_ = repo.CreateArticle(&model.Article{
			ID:     4,
			Status: model.Deleted, // 已处于物理/逻辑删除态
		})

		err := service.UpdateArticle(&model.Article{ID: 4})
		if err != common.ErrArticleDeleted {
			t.Errorf("对已删除文章操作，预期熔断返回 ErrArticleDeleted")
		}
	})
}

// 3. 测试获取文章详情
func TestArticleService_GetArticle(t *testing.T) {
	repo := repository.NewArticleRepository()
	service := NewArticleService(repo)

	// 正常发布的文章
	_ = repo.CreateArticle(&model.Article{ID: 5, Title: "公开文章", Status: model.Published})
	// 逻辑删除的文章
	_ = repo.CreateArticle(&model.Article{ID: 6, Title: "回收站文章", Status: model.Deleted})

	t.Run("正常获取", func(t *testing.T) {
		art, err := service.GetArticle(5)
		if err != nil || art.Title != "公开文章" {
			t.Errorf("未能成功获取合法文章")
		}
	})

	t.Run("不存在的ID", func(t *testing.T) {
		_, err := service.GetArticle(9999) // 随便编一个ID
		if err == nil {
			t.Errorf("应该返回未找到错误")
		}
	})

	t.Run("已被删除的拦截", func(t *testing.T) {
		_, err := service.GetArticle(6)
		if err != common.ErrArticleDeleted {
			t.Errorf("被删除的文章对外部应该触发 ErrArticleDeleted 熔断")
		}
	})
}

// 4. 测试发表文章状态转换
func TestArticleService_PublishArticle(t *testing.T) {
	repo := repository.NewArticleRepository()
	service := NewArticleService(repo)

	_ = repo.CreateArticle(&model.Article{
		ID:       7,
		AuthorID: 100,
		Status:   model.Draft,
	})

	err := service.PublishArticle(7, 100)
	if err != nil {
		t.Fatalf("发表文章动作失败: %v", err)
	}

	// 再次取出断言状态
	art, _ := repo.FindArticleByID(7)
	if art.Status != model.Published {
		t.Errorf("文章状态未能从 Draft 转换为 Published")
	}
}

// 5. 测试删除文章
func TestArticleService_DeleteArticle(t *testing.T) {
	repo := repository.NewArticleRepository()
	service := NewArticleService(repo)

	_ = repo.CreateArticle(&model.Article{ID: 8, AuthorID: 100, Status: model.Published})

	err := service.DeleteArticle(8, 100)
	if err != nil {
		t.Fatalf("调用删除失败: %v", err)
	}
}

// 6. 测试列表过滤 (已发布/草稿)
func TestArticleService_GetLists(t *testing.T) {
	repo := repository.NewArticleRepository()
	service := NewArticleService(repo)

	authorID := int64(100)
	_ = repo.CreateArticle(&model.Article{ID: 11, AuthorID: authorID, Status: model.Published})
	_ = repo.CreateArticle(&model.Article{ID: 12, AuthorID: authorID, Status: model.Draft})
	_ = repo.CreateArticle(&model.Article{ID: 13, AuthorID: authorID, Status: model.Draft})

	t.Run("获取已发表", func(t *testing.T) {
		list, _ := service.GetPublishedList(authorID)
		if len(list) != 1 {
			t.Errorf("预期已发布 1 篇，实际得到 %d 篇", len(list))
		}
	})

	// t.Run("获取草稿箱", func(t *testing.T) {
	// 	list, _ := service.GetDraftedList(authorID)
	// 	if len(list) != 2 {
	// 		t.Errorf("预期草稿 2 篇，实际得到 %d 篇", len(list))
	// 	}
	// })
}
