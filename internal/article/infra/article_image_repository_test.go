package infra

import (
	domaincontent "blog/internal/article/domain"
	"blog/internal/article/infra/model"
	platformtransaction "blog/internal/platform/transaction"
	"context"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newArticleImageRepositoryTestDB 创建文章图片 Repository 测试数据库。
func newArticleImageRepositoryTestDB(t *testing.T) *gorm.DB {
	// 1. 创建独立内存数据库并迁移图片表
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.ArticleImage{}); err != nil {
		t.Fatalf("迁移文章图片表失败: %v", err)
	}
	return db
}

// TestArticleImageRepositoryLifecycle 验证图片创建、绑定、查询、解绑和删除流程。
func TestArticleImageRepositoryLifecycle(t *testing.T) {
	// 1. 创建未绑定图片并验证数据库回填
	ctx := context.Background()
	repo := NewArticleImageRepository(newArticleImageRepositoryTestDB(t))
	image := &domaincontent.ArticleImage{ObjectKey: "article/img/2026/08/a.png"}
	if err := repo.Create(ctx, image); err != nil {
		t.Fatalf("创建图片记录失败: %v", err)
	}
	if image.ID == 0 || image.CreatedTime.IsZero() {
		t.Fatalf("图片数据库字段未回填: %+v", image)
	}

	// 2. 绑定图片并验证文章过滤查询
	rows, err := repo.BindArticle(ctx, []uint64{image.ID}, 10)
	if err != nil || rows != 1 {
		t.Fatalf("绑定图片失败: rows=%d err=%v", rows, err)
	}
	bound, err := repo.FindByArticleIDAndIDs(ctx, 10, []uint64{image.ID})
	if err != nil || len(bound) != 1 || bound[0].ArticleID != 10 {
		t.Fatalf("查询绑定图片错误: images=%+v err=%v", bound, err)
	}
	rows, err = repo.BindArticle(ctx, []uint64{image.ID}, 20)
	if err != nil || rows != 0 {
		t.Fatalf("已绑定图片不应被覆盖: rows=%d err=%v", rows, err)
	}

	// 3. 解绑图片后按文章删除不应命中
	rows, err = repo.UnbindArticle(ctx, []uint64{image.ID}, 10)
	if err != nil || rows != 1 {
		t.Fatalf("解绑图片失败: rows=%d err=%v", rows, err)
	}
	images, err := repo.FindByArticleID(ctx, 10)
	if err != nil || len(images) != 0 {
		t.Fatalf("解绑后仍查询到图片: images=%+v err=%v", images, err)
	}
	rows, err = repo.DeleteByArticleID(ctx, 10)
	if err != nil || rows != 0 {
		t.Fatalf("空文章图片删除结果错误: rows=%d err=%v", rows, err)
	}
}

// TestArticleImageRepositoryEmptyAndMissingQueries 验证空集合和缺失记录查询。
func TestArticleImageRepositoryEmptyAndMissingQueries(t *testing.T) {
	// 1. 空集合和不存在 ID 均返回空结果
	ctx := context.Background()
	repo := NewArticleImageRepository(newArticleImageRepositoryTestDB(t))
	for name, ids := range map[string][]uint64{"empty": {}, "missing": {999}} {
		images, err := repo.FindByIDs(ctx, ids)
		if err != nil || len(images) != 0 {
			t.Fatalf("%s 查询结果错误: images=%+v err=%v", name, images, err)
		}
	}
}

// TestArticleImageRepositoryUsesTransactionContext 验证图片写入复用并回滚事务上下文。
func TestArticleImageRepositoryUsesTransactionContext(t *testing.T) {
	// 1. 在事务内创建图片并主动返回错误
	ctx := context.Background()
	db := newArticleImageRepositoryTestDB(t)
	repo := NewArticleImageRepository(db)
	manager, err := platformtransaction.NewManager(db)
	if err != nil {
		t.Fatalf("创建事务协调器失败: %v", err)
	}
	rollbackErr := errors.New("rollback image transaction")
	var imageID uint64
	err = manager.WithinTransaction(ctx, func(txCtx context.Context) error {
		image := &domaincontent.ArticleImage{ObjectKey: "article/img/2026/08/rollback.png"}
		if err := repo.Create(txCtx, image); err != nil {
			return err
		}
		imageID = image.ID
		return rollbackErr
	})
	if !errors.Is(err, rollbackErr) {
		t.Fatalf("事务错误不正确: %v", err)
	}

	// 2. 事务回滚后默认连接查询不到图片
	images, err := repo.FindByIDs(ctx, []uint64{imageID})
	if err != nil || len(images) != 0 {
		t.Fatalf("回滚后仍存在图片记录: images=%+v err=%v", images, err)
	}
}
