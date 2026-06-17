package repository

import (
	"blog/internal/model"
	"testing"
)

func TestArticleRepository_CreateAndFind(t *testing.T) {
	repo := NewArticleRepository()

	article := &model.Article{
		ID:       1,
		AuthorID: 100,
		Title:    "test",
		Status:   model.Draft,
	}

	err := repo.CreateArticle(article)
	if err != nil {
		t.Fatalf("创建文章失败: %v", err)
	}

	res, err := repo.FindArticleByID(1)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}

	if res.Title != "test" {
		t.Fatalf("文章标题不一致")
	}
}

func TestArticleRepository_Update(t *testing.T) {
	repo := NewArticleRepository()

	article := &model.Article{
		ID:       2,
		AuthorID: 200,
		Title:    "old",
		Status:   model.Draft,
	}

	_ = repo.CreateArticle(article)

	article.Title = "new"
	err := repo.UpdateArticle(article)
	if err != nil {
		t.Fatalf("更新失败: %v", err)
	}

	res, _ := repo.FindArticleByID(2)
	if res.Title != "new" {
		t.Fatalf("更新未生效")
	}
}

func TestArticleRepository_Delete(t *testing.T) {
	repo := NewArticleRepository()

	article := &model.Article{
		ID:       3,
		AuthorID: 300,
		Status:   model.Published,
	}

	_ = repo.CreateArticle(article)

	err := repo.DeleteArticle(3, 300)
	if err != nil {
		t.Fatalf("删除失败: %v", err)
	}

	res, _ := repo.FindArticleByID(3)
	if res.Status != model.Deleted {
		t.Fatalf("软删除失败")
	}
}

func TestArticleRepository_GetListByStatus(t *testing.T) {
	repo := NewArticleRepository()

	repo.CreateArticle(&model.Article{ID: 1, AuthorID: 1, Status: model.Published})
	repo.CreateArticle(&model.Article{ID: 2, AuthorID: 1, Status: model.Draft})
	repo.CreateArticle(&model.Article{ID: 3, AuthorID: 2, Status: model.Published})

	list, _ := repo.GetListByStatus(1, model.Published)

	if len(list) != 1 {
		t.Fatalf("列表过滤失败")
	}
}
