package domain

import "testing"

func TestArticleLifecycleRules(t *testing.T) {
	article := &Article{ID: 1, AuthorID: 100, Status: StatusDraft}

	if article.IsPubliclyVisible() {
		t.Fatal("草稿不应公开可见")
	}
	if !article.CanEdit(100) || article.CanEdit(200) {
		t.Fatal("作者权限规则错误")
	}
	if !article.CanPublish(100) || article.CanDelete(200) {
		t.Fatal("发布/删除权限规则错误")
	}

	article.Publish()
	if !article.IsPublished() || !article.IsPubliclyVisible() {
		t.Fatal("发布后应公开可见")
	}

	article.SoftDelete()
	if !article.IsDeleted() || article.IsPubliclyVisible() {
		t.Fatal("删除后不可公开可见")
	}

	article.Recover()
	if !article.IsDraft() || article.IsDeleted() {
		t.Fatal("恢复后应回到草稿")
	}
}

func TestArticleStatusConstants(t *testing.T) {
	if StatusDeleted != 1 || StatusDraft != 2 || StatusPublished != 3 {
		t.Fatalf("文章状态常量被改变: %d %d %d", StatusDeleted, StatusDraft, StatusPublished)
	}
}
