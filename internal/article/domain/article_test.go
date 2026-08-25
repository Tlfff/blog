package domain

import (
	"errors"
	"testing"
)

// TestNewArticle 验证文章构造规则和兼容状态值。
func TestNewArticle(t *testing.T) {
	article, err := NewArticle(100, "标题", "正文", "Go", StatusDraft.Int8())
	if err != nil {
		t.Fatalf("创建文章失败: %v", err)
	}
	if article.AuthorID != 100 || !article.IsDraft() {
		t.Fatalf("文章初始化错误: %+v", article)
	}
	if _, err := NewArticle(100, "", "正文", "", StatusDraft.Int8()); !errors.Is(err, ErrArticleTitleEmpty) {
		t.Fatalf("空标题错误不正确: %v", err)
	}
	if _, err := NewArticle(100, "标题", "正文", "", 99); !errors.Is(err, ErrArticleStatusInvalid) {
		t.Fatalf("非法状态错误不正确: %v", err)
	}
}

// TestNewDraftArticle 验证初始化草稿允许空内容且保持作者和状态约束。
func TestNewDraftArticle(t *testing.T) {
	// 1. 验证合法作者可以初始化空内容草稿
	article, err := NewDraftArticle(100)
	if err != nil {
		t.Fatalf("初始化草稿失败: %v", err)
	}
	if article.AuthorID != 100 || !article.IsDraft() || article.Title != "" || article.Content != "" {
		t.Fatalf("初始化草稿字段错误: %+v", article)
	}

	// 2. 验证无效作者被领域规则拒绝
	if _, err := NewDraftArticle(0); !errors.Is(err, ErrArticlePermissionDenied) {
		t.Fatalf("无效作者错误不正确: %v", err)
	}
}

// TestArticleImageUploadPermission 验证文章图片上传权限和删除状态规则。
func TestArticleImageUploadPermission(t *testing.T) {
	// 1. 验证作者可以为活动文章上传图片
	article := &Article{ID: 1, AuthorID: 100, Status: StatusDraft}
	if err := article.EnsureCanUploadImageBy(100); err != nil {
		t.Fatalf("作者应允许上传图片: %v", err)
	}

	// 2. 验证非作者被拒绝
	if err := article.EnsureCanUploadImageBy(200); !errors.Is(err, ErrArticlePermissionDenied) {
		t.Fatalf("非作者上传错误不正确: %v", err)
	}

	// 3. 验证已删除文章不能继续上传图片
	article.Status = StatusDeleted
	if err := article.EnsureCanUploadImageBy(100); !errors.Is(err, ErrArticleDeleted) {
		t.Fatalf("已删除文章上传错误不正确: %v", err)
	}
}

// TestArticleLifecycleRules 验证文章生命周期由聚合保护。
func TestArticleLifecycleRules(t *testing.T) {
	article := &Article{ID: 1, AuthorID: 100, Status: StatusDraft}

	if err := article.PublishBy(200); !errors.Is(err, ErrArticlePermissionDenied) {
		t.Fatalf("非作者发布错误不正确: %v", err)
	}
	if err := article.PublishBy(100); err != nil || !article.IsPublished() {
		t.Fatalf("作者发布失败: %v", err)
	}
	if err := article.MoveToTrashBy(100); err != nil || !article.IsDeleted() {
		t.Fatalf("移入垃圾箱失败: %v", err)
	}
	if err := article.PublishBy(100); !errors.Is(err, ErrArticleDeleted) {
		t.Fatalf("已删除文章发布错误不正确: %v", err)
	}
	if err := article.RecoverBy(100); err != nil || !article.IsDraft() {
		t.Fatalf("恢复文章失败: %v", err)
	}
	if article.AuthorID != 100 {
		t.Fatalf("恢复文章不应改变作者: %d", article.AuthorID)
	}
}

// TestPermanentDeleteRequiresTrash 验证彻底删除必须先进入垃圾箱。
func TestPermanentDeleteRequiresTrash(t *testing.T) {
	article := &Article{ID: 1, AuthorID: 100, Status: StatusPublished}
	if err := article.EnsureCanPermanentlyDeleteBy(100); !errors.Is(err, ErrArticleStatusError) {
		t.Fatalf("活动文章应拒绝彻底删除: %v", err)
	}
	article.Status = StatusDeleted
	if err := article.EnsureCanPermanentlyDeleteBy(100); err != nil {
		t.Fatalf("垃圾箱文章应允许彻底删除: %v", err)
	}
}

// TestArticleStatusConstants 验证文章状态值保持兼容。
func TestArticleStatusConstants(t *testing.T) {
	if StatusDeleted != 1 || StatusDraft != 2 || StatusPublished != 3 {
		t.Fatalf("文章状态常量被改变: %d %d %d", StatusDeleted, StatusDraft, StatusPublished)
	}
}
