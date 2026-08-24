package app

import (
	domaincontent "blog/internal/article/domain"
	apperrors "blog/internal/shared/apperrors"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// fakeArticleRepo 是 Article Application 测试使用的内存 Repository。
type fakeArticleRepo struct {
	articles map[uint64]*domaincontent.Article // 文章唯一标识到文章聚合的映射
	nextID   uint64                            // 下一条文章记录的自增标识
}

// newFakeArticleRepo 创建内存文章 Repository。
func newFakeArticleRepo() *fakeArticleRepo {
	return &fakeArticleRepo{articles: make(map[uint64]*domaincontent.Article), nextID: 1}
}

// Create 保存文章并回填唯一标识与时间。
func (f *fakeArticleRepo) Create(_ context.Context, article *domaincontent.Article) error {
	article.ID = f.nextID
	f.nextID++
	now := time.Now()
	article.CreatedTime = now
	article.UpdatedTime = now
	f.articles[article.ID] = cloneArticle(article)
	return nil
}

// FindByID 根据唯一标识查询文章。
func (f *fakeArticleRepo) FindByID(_ context.Context, id uint64) (*domaincontent.Article, error) {
	article, ok := f.articles[id]
	if !ok {
		return nil, domaincontent.ErrArticleNotFound
	}
	return cloneArticle(article), nil
}

// FindWithAuthorByID 查询文章及测试作者展示信息。
func (f *fakeArticleRepo) FindWithAuthorByID(_ context.Context, id uint64) (*domaincontent.ArticleWithAuthor, error) {
	article, err := f.FindByID(context.Background(), id)
	if err != nil {
		return nil, err
	}
	return &domaincontent.ArticleWithAuthor{
		Article:     *article,
		Nickname:    "作者",
		Avatar:      "avatar",
		LastLoginIP: "127.0.0.1",
	}, nil
}

// Update 覆盖保存文章聚合。
func (f *fakeArticleRepo) Update(_ context.Context, article *domaincontent.Article) error {
	f.articles[article.ID] = cloneArticle(article)
	return nil
}

// SoftDelete 将文章状态改为已删除。
func (f *fakeArticleRepo) SoftDelete(_ context.Context, articleID uint64) error {
	article, ok := f.articles[articleID]
	if !ok {
		return domaincontent.ErrArticleNotFound
	}
	article.Status = domaincontent.StatusDeleted
	return nil
}

// Clear 物理删除文章记录。
func (f *fakeArticleRepo) Clear(_ context.Context, articleID uint64) error {
	delete(f.articles, articleID)
	return nil
}

// ListWithCursor 使用游标查询文章列表。
//
// 参数说明：
//   - ctx：测试上下文，本实现不使用。
//   - lastID：游标文章唯一标识。
//   - pageSize：每页数量。
//   - isDesc：是否倒序排列。
//   - status：文章状态过滤值。
func (f *fakeArticleRepo) ListWithCursor(_ context.Context, lastID uint64, pageSize int, isDesc bool, status int8) ([]*domaincontent.Article, error) {
	ids := make([]uint64, 0)
	for id := range f.articles {
		ids = append(ids, id)
	}
	sortUint64(ids)
	result := make([]*domaincontent.Article, 0)
	for _, id := range ids {
		article := f.articles[id]
		if article.Status.Int8() != status {
			continue
		}
		if isDesc && id < lastID {
			result = append(result, cloneArticle(article))
		}
		if !isDesc && id > lastID {
			result = append(result, cloneArticle(article))
		}
	}
	if isDesc {
		reverse(result)
	}
	if len(result) > pageSize {
		result = result[:pageSize]
	}
	return result, nil
}

// ListWithOffset 使用 Offset 查询文章列表。
//
// 参数说明：
//   - ctx：测试上下文，本实现不使用。
//   - page：当前页码。
//   - pageSize：每页数量。
//   - isDesc：是否倒序排列。
//   - status：文章状态过滤值。
func (f *fakeArticleRepo) ListWithOffset(_ context.Context, page, pageSize int, isDesc bool, status int8) ([]*domaincontent.Article, error) {
	all, err := f.ListWithCursor(context.Background(), 0, 1<<30, isDesc, status)
	if err != nil {
		return nil, err
	}
	start := (page - 1) * pageSize
	if start >= len(all) {
		return []*domaincontent.Article{}, nil
	}
	end := start + pageSize
	if end > len(all) {
		end = len(all)
	}
	return all[start:end], nil
}

// CountByStatus 按状态统计文章数量。
func (f *fakeArticleRepo) CountByStatus(_ context.Context, status int8) (int64, error) {
	var count int64
	for _, article := range f.articles {
		if article.Status.Int8() == status {
			count++
		}
	}
	return count, nil
}

// fakeImageStorage 是 Article Application 测试使用的对象存储。
type fakeImageStorage struct {
	moved []string // 已执行的对象移动记录
}

// PresignedPutURL 返回测试上传地址。
func (f *fakeImageStorage) PresignedPutURL(_ context.Context, objectKey string, _ time.Duration) (string, error) {
	return "https://upload.example/" + objectKey, nil
}

// GetObjectURL 返回测试对象访问地址。
func (f *fakeImageStorage) GetObjectURL(publicDomain, objectKey string) string {
	return publicDomain + "/" + objectKey
}

// MoveObject 记录对象移动操作。
func (f *fakeImageStorage) MoveObject(_ context.Context, srcKey, dstKey string) error {
	f.moved = append(f.moved, srcKey+"->"+dstKey)
	return nil
}

// TestContentService_ArticleLifecycle 验证文章创建、发布、权限和软删除流程。
func TestContentService_ArticleLifecycle(t *testing.T) {
	repo := newFakeArticleRepo()
	s := NewService(repo, nil, &fakeImageStorage{}, nil, "https://cdn.example", []string{"png", "jpg"})
	ctx := context.Background()

	if err := s.CreateArticle(ctx, 100, "草稿", "内容", []string{"Go"}, domaincontent.StatusDraft.Int8()); err != nil {
		t.Fatalf("创建草稿失败: %v", err)
	}
	if _, err := s.GetPublishedArticle(ctx, 1, 0); !errors.Is(err, apperrors.ErrArticlePermissionDenied) {
		t.Fatalf("草稿不应公开可见, got %v", err)
	}
	if err := s.PublishArticle(ctx, 1, 100); err != nil {
		t.Fatalf("发布失败: %v", err)
	}
	detail, err := s.GetPublishedArticle(ctx, 1, 0)
	if err != nil {
		t.Fatalf("已发布文章应可读: %v", err)
	}
	if detail.ID != 1 || detail.Title != "草稿" {
		t.Fatalf("文章详情数据不一致: %+v", detail)
	}
	if err := s.UpdateArticle(ctx, 1, 999, "x", "y", nil, domaincontent.StatusDraft.Int8()); !errors.Is(err, apperrors.ErrArticlePermissionDenied) {
		t.Fatalf("非作者更新应被拒绝, got %v", err)
	}
	if err := s.DeleteArticle(ctx, 1, 100); err != nil {
		t.Fatalf("删除失败: %v", err)
	}
	if _, err := s.GetPublishedArticle(ctx, 1, 0); !errors.Is(err, apperrors.ErrArticleDeleted) {
		t.Fatalf("已删除文章应返回删除错误, got %v", err)
	}
}

// TestContentService_ClearRequiresDeletedAndRecoverKeepsAuthor 验证彻底删除和恢复规则。
func TestContentService_ClearRequiresDeletedAndRecoverKeepsAuthor(t *testing.T) {
	repo := newFakeArticleRepo()
	service := NewService(repo, nil, &fakeImageStorage{}, nil, "https://cdn.example", nil)
	ctx := context.Background()

	if err := service.CreateArticle(ctx, 100, "文章", "内容", nil, domaincontent.StatusPublished.Int8()); err != nil {
		t.Fatalf("创建文章失败: %v", err)
	}
	if err := service.ClearArticle(ctx, 1, 100); !errors.Is(err, apperrors.ErrArticleStatusError) {
		t.Fatalf("活动文章应拒绝彻底删除: %v", err)
	}
	if _, err := repo.FindByID(ctx, 1); err != nil {
		t.Fatalf("拒绝彻底删除后文章不应丢失: %v", err)
	}

	if err := service.DeleteArticle(ctx, 1, 100); err != nil {
		t.Fatalf("移入垃圾箱失败: %v", err)
	}
	if err := service.RecoverArticle(ctx, 1, 100); err != nil {
		t.Fatalf("恢复文章失败: %v", err)
	}
	recovered, err := repo.FindByID(ctx, 1)
	if err != nil {
		t.Fatalf("查询恢复文章失败: %v", err)
	}
	if recovered.AuthorID != 100 || !recovered.IsDraft() {
		t.Fatalf("恢复文章不应改变作者且应回到草稿: %+v", recovered)
	}

	if err := service.DeleteArticle(ctx, 1, 100); err != nil {
		t.Fatalf("再次移入垃圾箱失败: %v", err)
	}
	if err := service.ClearArticle(ctx, 1, 100); err != nil {
		t.Fatalf("彻底删除垃圾箱文章失败: %v", err)
	}
	if _, err := repo.FindByID(ctx, 1); !errors.Is(err, domaincontent.ErrArticleNotFound) {
		t.Fatalf("彻底删除后文章仍存在: %v", err)
	}
}

// TestContentService_PublishedListAndAvailable 验证公开文章列表过滤规则。
func TestContentService_PublishedListAndAvailable(t *testing.T) {
	repo := newFakeArticleRepo()
	s := NewService(repo, nil, &fakeImageStorage{}, nil, "https://cdn.example", nil)
	ctx := context.Background()

	_ = s.CreateArticle(ctx, 1, "A", "内容", nil, domaincontent.StatusPublished.Int8())
	_ = s.CreateArticle(ctx, 1, "B", "内容", nil, domaincontent.StatusDraft.Int8())
	_ = s.CreateArticle(ctx, 1, "C", "内容", nil, domaincontent.StatusPublished.Int8())

	list, err := s.GetPublishedList(ctx, 1, 10, 0, false)
	if err != nil {
		t.Fatalf("获取公开列表失败: %v", err)
	}
	if list.Total != 2 || len(list.List) != 2 {
		t.Fatalf("公开列表状态过滤错误: total=%d len=%d", list.Total, len(list.List))
	}
	if list.LastID != list.List[len(list.List)-1].ID {
		t.Fatalf("游标 last_id 不一致: %d", list.LastID)
	}
	external, err := s.GetAvailableList(ctx, 1, 10, false)
	if err != nil {
		t.Fatalf("获取开放列表失败: %v", err)
	}
	if external.Total != 2 {
		t.Fatalf("开放列表应排除草稿与删除, total=%d", external.Total)
	}
}

// TestContentService_ImageContract 验证文章图片上传和转正契约。
func TestContentService_ImageContract(t *testing.T) {
	storage := &fakeImageStorage{}
	s := NewService(newFakeArticleRepo(), nil, storage, nil, "https://cdn.example", []string{"png", "jpg"})
	ctx := context.Background()

	if _, _, err := s.GetUploadURL(ctx, "exe"); !errors.Is(err, apperrors.ErrInvalidRequestBody) {
		t.Fatalf("非法扩展名应被拒绝, got %v", err)
	}
	uploadURL, url, err := s.GetUploadURL(ctx, "png")
	if err != nil {
		t.Fatalf("获取上传凭证失败: %v", err)
	}
	if !strings.HasPrefix(uploadURL, "https://upload.example/article/temp/") || !strings.HasSuffix(url, ".png") {
		t.Fatalf("上传凭证契约被改变: %s %s", uploadURL, url)
	}

	content := "![img](https://cdn.example/article/temp/abc.png)"
	promoted, err := s.PromoteImages(ctx, 7, content)
	if err != nil {
		t.Fatalf("图片转正失败: %v", err)
	}
	if !strings.Contains(promoted, "https://cdn.example/article/7/abc.png") || strings.Contains(promoted, "/temp/") {
		t.Fatalf("图片转正 URL 错误: %s", promoted)
	}
	if len(storage.moved) != 1 || storage.moved[0] != "article/temp/abc.png->article/7/abc.png" {
		t.Fatalf("对象移动记录错误: %v", storage.moved)
	}
}

// cloneArticle 复制文章聚合，避免测试数据被调用方直接修改。
func cloneArticle(a *domaincontent.Article) *domaincontent.Article {
	if a == nil {
		return nil
	}
	c := *a
	return &c
}

// sortUint64 对无符号整数切片进行升序排序。
func sortUint64(values []uint64) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j-1] > values[j]; j-- {
			values[j-1], values[j] = values[j], values[j-1]
		}
	}
}

// reverse 反转文章切片顺序。
func reverse(articles []*domaincontent.Article) {
	for i, j := 0, len(articles)-1; i < j; i, j = i+1, j-1 {
		articles[i], articles[j] = articles[j], articles[i]
	}
}
