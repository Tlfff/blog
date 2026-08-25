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
	articles  map[uint64]*domaincontent.Article // 文章唯一标识到文章聚合的映射
	nextID    uint64                            // 下一条文章记录的自增标识
	createErr error                             // 创建文章时返回的测试错误
	clearErr  error                             // 物理删除文章时返回的测试错误
	clearCall int                               // 物理删除调用次数
}

// newFakeArticleRepo 创建内存文章 Repository。
func newFakeArticleRepo() *fakeArticleRepo {
	return &fakeArticleRepo{articles: make(map[uint64]*domaincontent.Article), nextID: 1}
}

// Create 保存文章并回填唯一标识与时间。
func (f *fakeArticleRepo) Create(_ context.Context, article *domaincontent.Article) error {
	if f.createErr != nil {
		return f.createErr
	}
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
	f.clearCall++
	if f.clearErr != nil {
		return f.clearErr
	}
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
	presignedKeys   []string // 已签发上传凭证的对象 Key
	deletedPrefixes []string // 已执行的对象前缀删除记录
	presignErr      error    // 生成上传凭证时返回的测试错误
	deleteErr       error    // 删除对象前缀时返回的测试错误
}

// PresignedPutURL 返回测试上传地址。
func (f *fakeImageStorage) PresignedPutURL(_ context.Context, objectKey string, _ time.Duration) (string, error) {
	if f.presignErr != nil {
		return "", f.presignErr
	}
	f.presignedKeys = append(f.presignedKeys, objectKey)
	return "https://upload.example/" + objectKey, nil
}

// GetObjectURL 返回测试对象访问地址。
func (f *fakeImageStorage) GetObjectURL(publicDomain, objectKey string) string {
	return publicDomain + "/" + objectKey
}

// DeleteObjectsByPrefix 记录对象前缀删除操作。
func (f *fakeImageStorage) DeleteObjectsByPrefix(_ context.Context, prefix string) error {
	// 1. 记录前缀并返回预设测试错误
	f.deletedPrefixes = append(f.deletedPrefixes, prefix)
	return f.deleteErr
}

// TestContentService_InitializeArticle 验证初始化空草稿和 Repository 失败路径。
func TestContentService_InitializeArticle(t *testing.T) {
	// 1. 验证初始化文章保存空内容草稿并返回 ID
	repo := newFakeArticleRepo()
	service := NewService(repo, nil, &fakeImageStorage{}, nil, "https://cdn.example", nil)

	result, err := service.InitializeArticle(context.Background(), 100)
	if err != nil {
		t.Fatalf("初始化文章失败: %v", err)
	}
	article, err := repo.FindByID(context.Background(), result.ArticleID)
	if err != nil {
		t.Fatalf("查询初始化文章失败: %v", err)
	}
	if article.AuthorID != 100 || !article.IsDraft() || article.Title != "" || article.Content != "" {
		t.Fatalf("初始化文章字段错误: %+v", article)
	}

	// 2. 验证 Repository 创建错误原样返回
	repo.createErr = errors.New("create failed")
	if _, err := service.InitializeArticle(context.Background(), 100); !errors.Is(err, repo.createErr) {
		t.Fatalf("创建失败错误未透传: %v", err)
	}
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

// TestContentService_BatchImageUploadURLs 验证批量凭证、权限和正式对象路径规则。
func TestContentService_BatchImageUploadURLs(t *testing.T) {
	// 1. 初始化文章并为不同扩展名图片获取正式路径凭证
	ctx := context.Background()
	repo := newFakeArticleRepo()
	storage := &fakeImageStorage{}
	service := NewService(repo, nil, storage, nil, "https://cdn.example", []string{"png", "jpg"})
	initialized, err := service.InitializeArticle(ctx, 100)
	if err != nil {
		t.Fatalf("初始化文章失败: %v", err)
	}

	result, err := service.GetImageUploadURLs(ctx, GetImageUploadURLsCommand{
		ArticleID: initialized.ArticleID,
		AuthorID:  100,
		Files: []ImageUploadFileCommand{
			{ClientID: "image-1", FileExt: "png"},
			{ClientID: "image-2", FileExt: ".JPG"},
		},
	})
	if err != nil {
		t.Fatalf("批量获取上传凭证失败: %v", err)
	}
	if len(result.Files) != 2 || result.Files[0].ClientID != "image-1" || result.Files[1].ClientID != "image-2" {
		t.Fatalf("批量凭证对应关系错误: %+v", result.Files)
	}
	for index, key := range storage.presignedKeys {
		if !strings.HasPrefix(key, "article/1/") || strings.Contains(key, "/temp/") {
			t.Fatalf("第 %d 个对象 Key 不是文章正式路径: %s", index, key)
		}
	}
	if !strings.HasSuffix(storage.presignedKeys[0], ".png") || !strings.HasSuffix(storage.presignedKeys[1], ".jpg") {
		t.Fatalf("不同扩展名未分别保留: %v", storage.presignedKeys)
	}

	// 2. 验证非法扩展名导致整批失败且不签发部分凭证
	storage.presignedKeys = nil
	if _, err := service.GetImageUploadURLs(ctx, GetImageUploadURLsCommand{
		ArticleID: initialized.ArticleID,
		AuthorID:  100,
		Files: []ImageUploadFileCommand{
			{ClientID: "valid", FileExt: "png"},
			{ClientID: "invalid", FileExt: "exe"},
		},
	}); !errors.Is(err, apperrors.ErrInvalidRequestBody) {
		t.Fatalf("非法扩展名错误不正确: %v", err)
	}
	if len(storage.presignedKeys) != 0 {
		t.Fatalf("非法批量请求不应签发部分凭证: %v", storage.presignedKeys)
	}

	// 3. 验证非作者和已删除文章均不能获取凭证
	validFiles := []ImageUploadFileCommand{{ClientID: "image", FileExt: "png"}}
	if _, err := service.GetImageUploadURLs(ctx, GetImageUploadURLsCommand{
		ArticleID: initialized.ArticleID,
		AuthorID:  200,
		Files:     validFiles,
	}); !errors.Is(err, apperrors.ErrArticlePermissionDenied) {
		t.Fatalf("非作者上传错误不正确: %v", err)
	}
	repo.articles[initialized.ArticleID].Status = domaincontent.StatusDeleted
	if _, err := service.GetImageUploadURLs(ctx, GetImageUploadURLsCommand{
		ArticleID: initialized.ArticleID,
		AuthorID:  100,
		Files:     validFiles,
	}); !errors.Is(err, apperrors.ErrArticleDeleted) {
		t.Fatalf("已删除文章上传错误不正确: %v", err)
	}
}

// TestContentService_BatchImageUploadStorageError 验证对象存储错误不会返回批量结果。
func TestContentService_BatchImageUploadStorageError(t *testing.T) {
	// 1. 初始化文章并配置对象存储签名错误
	ctx := context.Background()
	repo := newFakeArticleRepo()
	storageErr := errors.New("presign failed")
	storage := &fakeImageStorage{presignErr: storageErr}
	service := NewService(repo, nil, storage, nil, "https://cdn.example", []string{"png"})
	initialized, err := service.InitializeArticle(ctx, 100)
	if err != nil {
		t.Fatalf("初始化文章失败: %v", err)
	}

	// 2. 验证对象存储错误透传且不返回部分结果
	result, err := service.GetImageUploadURLs(ctx, GetImageUploadURLsCommand{
		ArticleID: initialized.ArticleID,
		AuthorID:  100,
		Files:     []ImageUploadFileCommand{{ClientID: "image", FileExt: "png"}},
	})
	if !errors.Is(err, storageErr) || result != nil {
		t.Fatalf("对象存储错误处理不正确: result=%+v err=%v", result, err)
	}
}

// TestContentService_NewImageFlowSavesFinalURL 验证新上传流程直接保存正式图片 URL。
func TestContentService_NewImageFlowSavesFinalURL(t *testing.T) {
	// 1. 初始化文章并获取直接写入正式目录的凭证
	ctx := context.Background()
	repo := newFakeArticleRepo()
	storage := &fakeImageStorage{}
	service := NewService(repo, nil, storage, nil, "https://cdn.example", []string{"png"})
	initialized, err := service.InitializeArticle(ctx, 100)
	if err != nil {
		t.Fatalf("初始化文章失败: %v", err)
	}
	credentials, err := service.GetImageUploadURLs(ctx, GetImageUploadURLsCommand{
		ArticleID: initialized.ArticleID,
		AuthorID:  100,
		Files:     []ImageUploadFileCommand{{ClientID: "image", FileExt: "png"}},
	})
	if err != nil {
		t.Fatalf("获取上传凭证失败: %v", err)
	}

	// 2. 使用正式 URL 更新文章并确认正文直接保存
	content := "![image](" + credentials.Files[0].URL + ")"
	if err := service.UpdateArticle(ctx, initialized.ArticleID, 100, "标题", content, nil, domaincontent.StatusDraft.Int8()); err != nil {
		t.Fatalf("完成初始化文章失败: %v", err)
	}
	article, err := repo.FindByID(ctx, initialized.ArticleID)
	if err != nil {
		t.Fatalf("查询更新后文章失败: %v", err)
	}
	if article.Content != content {
		t.Fatalf("文章未直接保存正式图片 URL: %s", article.Content)
	}
}

// TestContentService_ClearArticleImageCleanup 验证硬删除的图片清理顺序和重试语义。
func TestContentService_ClearArticleImageCleanup(t *testing.T) {
	// 1. 创建文章并移入垃圾箱
	ctx := context.Background()
	repo := newFakeArticleRepo()
	storage := &fakeImageStorage{}
	service := NewService(repo, nil, storage, nil, "https://cdn.example", nil)
	if err := service.CreateArticle(ctx, 100, "文章", "内容", nil, domaincontent.StatusDraft.Int8()); err != nil {
		t.Fatalf("创建文章失败: %v", err)
	}
	if err := service.DeleteArticle(ctx, 1, 100); err != nil {
		t.Fatalf("移入垃圾箱失败: %v", err)
	}

	// 2. 验证图片清理失败时不调用数据库物理删除
	storage.deleteErr = errors.New("delete images failed")
	if err := service.ClearArticle(ctx, 1, 100); !errors.Is(err, storage.deleteErr) {
		t.Fatalf("图片清理失败错误不正确: %v", err)
	}
	if repo.clearCall != 0 {
		t.Fatalf("图片清理失败时不应删除数据库记录: %d", repo.clearCall)
	}
	if _, err := repo.FindByID(ctx, 1); err != nil {
		t.Fatalf("图片清理失败后文章应保留: %v", err)
	}

	// 3. 验证图片清理成功但数据库删除失败时文章仍可重试
	storage.deleteErr = nil
	repo.clearErr = errors.New("clear article failed")
	if err := service.ClearArticle(ctx, 1, 100); !errors.Is(err, repo.clearErr) {
		t.Fatalf("数据库删除失败错误不正确: %v", err)
	}
	if len(storage.deletedPrefixes) != 2 || storage.deletedPrefixes[0] != "article/1/" {
		t.Fatalf("文章图片前缀清理记录错误: %v", storage.deletedPrefixes)
	}
	if _, err := repo.FindByID(ctx, 1); err != nil {
		t.Fatalf("数据库删除失败后文章应保留: %v", err)
	}

	// 4. 验证空目录重试可以最终删除文章
	repo.clearErr = nil
	if err := service.ClearArticle(ctx, 1, 100); err != nil {
		t.Fatalf("重试硬删除失败: %v", err)
	}
	if _, err := repo.FindByID(ctx, 1); !errors.Is(err, domaincontent.ErrArticleNotFound) {
		t.Fatalf("重试后文章仍存在: %v", err)
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
