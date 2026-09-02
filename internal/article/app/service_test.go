package app

import (
	domaincontent "blog/internal/article/domain"
	articleinfra "blog/internal/article/infra"
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
	updateErr error                             // 更新文章时返回的测试错误
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
	if f.updateErr != nil {
		return f.updateErr
	}
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
	presignedKeys []string         // 已签发上传凭证的对象 Key
	deletedKeys   []string         // 已执行删除的对象 Key
	presignErr    error            // 生成上传凭证时返回的测试错误
	deleteErr     error            // 删除对象时返回的测试错误
	deleteErrors  map[string]error // 指定对象 Key 对应的删除错误
}

// PresignedPutURL 返回测试上传地址。
func (f *fakeImageStorage) PresignedPutURL(_ context.Context, objectKey string, _ time.Duration) (string, error) {
	// 1. 返回预设错误或记录对象 Key
	if f.presignErr != nil {
		return "", f.presignErr
	}
	f.presignedKeys = append(f.presignedKeys, objectKey)
	return "https://upload.example/" + objectKey, nil
}

// GetObjectURL 返回测试对象访问地址。
func (f *fakeImageStorage) GetObjectURL(publicDomain, objectKey string) string {
	// 1. 使用测试公开域名拼接对象 Key
	return publicDomain + "/" + objectKey
}

// DeleteObject 记录单对象删除操作。
func (f *fakeImageStorage) DeleteObject(_ context.Context, objectKey string) error {
	// 1. 记录对象 Key 并返回指定或通用测试错误
	f.deletedKeys = append(f.deletedKeys, objectKey)
	if err := f.deleteErrors[objectKey]; err != nil {
		return err
	}
	return f.deleteErr
}

// fakeArticleImageRepo 是 Article Application 测试使用的图片 Repository。
type fakeArticleImageRepo struct {
	images     map[uint64]*domaincontent.ArticleImage // 图片唯一标识到图片记录的映射
	nextID     uint64                                 // 下一条图片记录的自增标识
	createErr  error                                  // 创建图片时返回的测试错误
	findErr    error                                  // 查询图片时返回的测试错误
	bindErr    error                                  // 绑定图片时返回的测试错误
	unbindErr  error                                  // 解绑图片时返回的测试错误
	deleteErr  error                                  // 删除图片记录时返回的测试错误
	bindRows   *int64                                 // 非空时覆盖绑定影响行数
	unbindRows *int64                                 // 非空时覆盖解绑影响行数
	deleteRows *int64                                 // 非空时覆盖删除影响行数
}

// newFakeArticleImageRepo 创建内存图片 Repository。
func newFakeArticleImageRepo() *fakeArticleImageRepo {
	// 1. 初始化图片映射和自增标识
	return &fakeArticleImageRepo{images: make(map[uint64]*domaincontent.ArticleImage), nextID: 1}
}

// Create 创建图片记录并回填唯一标识。
func (f *fakeArticleImageRepo) Create(_ context.Context, image *domaincontent.ArticleImage) error {
	// 1. 返回预设错误或保存图片副本
	if f.createErr != nil {
		return f.createErr
	}
	image.ID = f.nextID
	f.nextID++
	image.CreatedTime = time.Now()
	f.images[image.ID] = cloneArticleImage(image)
	return nil
}

// FindByIDs 按图片 ID 批量查询图片。
func (f *fakeArticleImageRepo) FindByIDs(_ context.Context, ids []uint64) ([]*domaincontent.ArticleImage, error) {
	// 1. 返回预设错误或按请求顺序查询图片
	if f.findErr != nil {
		return nil, f.findErr
	}
	images := make([]*domaincontent.ArticleImage, 0, len(ids))
	for _, id := range ids {
		if image, ok := f.images[id]; ok {
			images = append(images, cloneArticleImage(image))
		}
	}
	return images, nil
}

// FindByIDsForUpdate 锁定语义下批量查询图片。
func (f *fakeArticleImageRepo) FindByIDsForUpdate(ctx context.Context, ids []uint64) ([]*domaincontent.ArticleImage, error) {
	// 1. 复用内存批量查询结果
	return f.FindByIDs(ctx, ids)
}

// FindByArticleIDAndIDs 查询属于指定文章的正文图片。
func (f *fakeArticleImageRepo) FindByArticleIDAndIDs(_ context.Context, articleID uint64, ids []uint64) ([]*domaincontent.ArticleImage, error) {
	// 1. 同时按文章归属和图片 ID 过滤
	if f.findErr != nil {
		return nil, f.findErr
	}
	images := make([]*domaincontent.ArticleImage, 0, len(ids))
	for _, id := range ids {
		if image, ok := f.images[id]; ok && image.ArticleID == articleID {
			images = append(images, cloneArticleImage(image))
		}
	}
	return images, nil
}

// FindByArticleID 查询指定文章的全部图片。
func (f *fakeArticleImageRepo) FindByArticleID(_ context.Context, articleID uint64) ([]*domaincontent.ArticleImage, error) {
	// 1. 返回预设错误或筛选文章图片
	if f.findErr != nil {
		return nil, f.findErr
	}
	images := make([]*domaincontent.ArticleImage, 0)
	for _, image := range f.images {
		if image.ArticleID == articleID {
			images = append(images, cloneArticleImage(image))
		}
	}
	return images, nil
}

// BindArticle 批量绑定未归属图片。
func (f *fakeArticleImageRepo) BindArticle(_ context.Context, ids []uint64, articleID uint64) (int64, error) {
	// 1. 返回预设错误或更新未绑定图片
	if f.bindErr != nil {
		return 0, f.bindErr
	}
	var rows int64
	for _, id := range ids {
		if image, ok := f.images[id]; ok && image.ArticleID == 0 {
			image.ArticleID = articleID
			rows++
		}
	}
	if f.bindRows != nil {
		return *f.bindRows, nil
	}
	return rows, nil
}

// UnbindArticle 批量解绑当前文章图片。
func (f *fakeArticleImageRepo) UnbindArticle(_ context.Context, ids []uint64, articleID uint64) (int64, error) {
	// 1. 返回预设错误或清空当前文章关系
	if f.unbindErr != nil {
		return 0, f.unbindErr
	}
	var rows int64
	for _, id := range ids {
		if image, ok := f.images[id]; ok && image.ArticleID == articleID {
			image.ArticleID = 0
			rows++
		}
	}
	if f.unbindRows != nil {
		return *f.unbindRows, nil
	}
	return rows, nil
}

// DeleteByArticleID 删除指定文章的图片记录。
func (f *fakeArticleImageRepo) DeleteByArticleID(_ context.Context, articleID uint64) (int64, error) {
	// 1. 返回预设错误或删除全部匹配图片
	if f.deleteErr != nil {
		return 0, f.deleteErr
	}
	var rows int64
	for id, image := range f.images {
		if image.ArticleID == articleID {
			delete(f.images, id)
			rows++
		}
	}
	if f.deleteRows != nil {
		return *f.deleteRows, nil
	}
	return rows, nil
}

// fakeTransactionManager 为内存 Repository 提供回滚语义。
type fakeTransactionManager struct {
	articles *fakeArticleRepo      // 参与事务的文章 Repository
	images   *fakeArticleImageRepo // 参与事务的图片 Repository
}

// WithinTransaction 执行回调并在失败时恢复内存快照。
func (f fakeTransactionManager) WithinTransaction(ctx context.Context, callback func(context.Context) error) error {
	// 1. 复制事务开始前的文章和图片状态
	articleSnapshot := cloneArticleMap(f.articles.articles)
	articleNextID := f.articles.nextID
	imageSnapshot := cloneArticleImageMap(f.images.images)
	imageNextID := f.images.nextID

	// 2. 执行事务回调，失败时恢复全部状态
	if err := callback(ctx); err != nil {
		f.articles.articles = articleSnapshot
		f.articles.nextID = articleNextID
		f.images.images = imageSnapshot
		f.images.nextID = imageNextID
		return err
	}
	return nil
}

// newTestService 创建具备图片和事务依赖的 Article Application 测试服务。
func newTestService(repo *fakeArticleRepo, imageRepo *fakeArticleImageRepo, storage *fakeImageStorage, allowedExts []string) *Service {
	// 1. 使用真实 Markdown 解析器和内存事务组装服务
	service := NewService(ServiceDependencies{
		Articles: repo, ArticleImages: imageRepo, ImageReferences: articleinfra.NewMarkdownImageReferenceParser(),
		ImageStorage: storage, Transactions: fakeTransactionManager{articles: repo, images: imageRepo},
		PublicDomain: "https://cdn.example", AllowedExts: allowedExts,
	})
	service.now = func() time.Time { return time.Date(2026, time.August, 28, 0, 0, 0, 0, time.UTC) }
	return service
}

// TestContentServiceArticleLifecycle 验证文章直接创建、发布、权限和软删除流程。
func TestContentServiceArticleLifecycle(t *testing.T) {
	// 1. 直接创建草稿并发布
	ctx := context.Background()
	repo := newFakeArticleRepo()
	imageRepo := newFakeArticleImageRepo()
	service := newTestService(repo, imageRepo, &fakeImageStorage{}, []string{"png", "jpg"})
	if err := service.CreateArticle(ctx, 100, "草稿", "内容", []string{"Go"}, domaincontent.StatusDraft.Int8()); err != nil {
		t.Fatalf("创建文章失败: %v", err)
	}
	if err := service.PublishArticle(ctx, 1, 100); err != nil {
		t.Fatalf("发布文章失败: %v", err)
	}

	// 2. 非作者不能更新，作者可以移入垃圾箱
	if err := service.UpdateArticle(ctx, 1, 200, "标题", "新内容", nil, domaincontent.StatusDraft.Int8()); !errors.Is(err, apperrors.ErrArticlePermissionDenied) {
		t.Fatalf("非作者更新错误不正确: %v", err)
	}
	if err := service.DeleteArticle(ctx, 1, 100); err != nil {
		t.Fatalf("软删除文章失败: %v", err)
	}
	if _, err := service.GetArticle(ctx, 1, 100); !errors.Is(err, apperrors.ErrArticleDeleted) {
		t.Fatalf("已删除文章详情错误不正确: %v", err)
	}
}

// TestContentServiceClearRequiresDeletedAndRecoverKeepsImages 验证硬删除状态和恢复期间图片关系保持不变。
func TestContentServiceClearRequiresDeletedAndRecoverKeepsImages(t *testing.T) {
	// 1. 创建包含图片的文章并验证活动文章不能硬删除
	ctx := context.Background()
	repo := newFakeArticleRepo()
	imageRepo := newFakeArticleImageRepo()
	image := &domaincontent.ArticleImage{ObjectKey: "article/img/2026/08/a.png"}
	_ = imageRepo.Create(ctx, image)
	service := newTestService(repo, imageRepo, &fakeImageStorage{}, []string{"png"})
	content := "![a](image://1)"
	if err := service.CreateArticle(ctx, 100, "文章", content, nil, domaincontent.StatusDraft.Int8()); err != nil {
		t.Fatalf("创建文章失败: %v", err)
	}
	if err := service.ClearArticle(ctx, 1, 100); !errors.Is(err, apperrors.ErrArticleStatusError) {
		t.Fatalf("活动文章硬删除错误不正确: %v", err)
	}

	// 2. 软删除和恢复均保留图片绑定关系
	if err := service.DeleteArticle(ctx, 1, 100); err != nil {
		t.Fatalf("软删除文章失败: %v", err)
	}
	if imageRepo.images[1].ArticleID != 1 {
		t.Fatalf("软删除不应解绑图片: %+v", imageRepo.images[1])
	}
	if err := service.RecoverArticle(ctx, 1, 100); err != nil {
		t.Fatalf("恢复文章失败: %v", err)
	}
	if imageRepo.images[1].ArticleID != 1 || repo.articles[1].AuthorID != 100 {
		t.Fatalf("恢复后文章或图片关系错误: article=%+v image=%+v", repo.articles[1], imageRepo.images[1])
	}
}

// TestContentServicePublishedListAndAvailable 验证公开列表和对外列表分页。
func TestContentServicePublishedListAndAvailable(t *testing.T) {
	// 1. 创建两篇已发表文章
	ctx := context.Background()
	repo := newFakeArticleRepo()
	service := newTestService(repo, newFakeArticleImageRepo(), &fakeImageStorage{}, nil)
	for _, title := range []string{"A", "B"} {
		if err := service.CreateArticle(ctx, 100, title, "内容", nil, domaincontent.StatusPublished.Int8()); err != nil {
			t.Fatalf("创建已发表文章失败: %v", err)
		}
	}

	// 2. 验证公开列表和二方列表均返回数据
	published, err := service.GetPublishedList(ctx, 1, 10, 0, false)
	if err != nil || len(published.List) != 2 {
		t.Fatalf("公开文章列表错误: result=%+v err=%v", published, err)
	}
	available, err := service.GetAvailableList(ctx, 1, 10, false)
	if err != nil || len(available.List) != 2 {
		t.Fatalf("对外文章列表错误: result=%+v err=%v", available, err)
	}
}

// TestContentServiceGetImageUploadURL 验证单图片凭证、路径和失败场景。
func TestContentServiceGetImageUploadURL(t *testing.T) {
	// 1. 合法扩展名创建未绑定图片并返回年月路径
	ctx := context.Background()
	repo := newFakeArticleRepo()
	imageRepo := newFakeArticleImageRepo()
	storage := &fakeImageStorage{}
	service := newTestService(repo, imageRepo, storage, []string{"png", "jpg"})
	result, err := service.GetImageUploadURL(ctx, GetImageUploadURLCommand{FileExt: ".PNG"})
	if err != nil {
		t.Fatalf("获取图片上传凭证失败: %v", err)
	}
	if result.ImageID != 1 || !strings.HasPrefix(storage.presignedKeys[0], "article/img/2026/08/") || imageRepo.images[1].ArticleID != 0 {
		t.Fatalf("图片上传结果错误: result=%+v images=%+v keys=%v", result, imageRepo.images, storage.presignedKeys)
	}

	// 2. 非法扩展名和对象存储失败不创建图片记录
	if _, err := service.GetImageUploadURL(ctx, GetImageUploadURLCommand{FileExt: "exe"}); !errors.Is(err, apperrors.ErrInvalidRequestBody) {
		t.Fatalf("非法扩展名错误不正确: %v", err)
	}
	storage.presignErr = errors.New("presign failed")
	if _, err := service.GetImageUploadURL(ctx, GetImageUploadURLCommand{FileExt: "png"}); !errors.Is(err, storage.presignErr) {
		t.Fatalf("预签名错误未透传: %v", err)
	}
	if len(imageRepo.images) != 1 {
		t.Fatalf("预签名失败不应创建图片记录: %+v", imageRepo.images)
	}

	// 3. 数据库失败时不返回凭证
	storage.presignErr = nil
	imageRepo.createErr = errors.New("create image failed")
	if result, err := service.GetImageUploadURL(ctx, GetImageUploadURLCommand{FileExt: "png"}); !errors.Is(err, imageRepo.createErr) || result != nil {
		t.Fatalf("图片记录失败处理错误: result=%+v err=%v", result, err)
	}
}

// TestContentServiceCreateAndUpdateImageRelations 验证创建和更新时的图片绑定与解绑。
func TestContentServiceCreateAndUpdateImageRelations(t *testing.T) {
	// 1. 准备三张未绑定图片并创建引用前两张的文章
	ctx := context.Background()
	repo := newFakeArticleRepo()
	imageRepo := newFakeArticleImageRepo()
	for _, key := range []string{"a.png", "b.png", "c.png"} {
		_ = imageRepo.Create(ctx, &domaincontent.ArticleImage{ObjectKey: "article/img/2026/08/" + key})
	}
	service := newTestService(repo, imageRepo, &fakeImageStorage{}, []string{"png"})
	content := "![a](image://1)\n![b](image://2)"
	if err := service.CreateArticle(ctx, 100, "文章", content, nil, domaincontent.StatusDraft.Int8()); err != nil {
		t.Fatalf("创建包含图片的文章失败: %v", err)
	}
	if imageRepo.images[1].ArticleID != 1 || imageRepo.images[2].ArticleID != 1 || imageRepo.images[3].ArticleID != 0 {
		t.Fatalf("创建时图片绑定错误: %+v", imageRepo.images)
	}

	// 2. 更新正文时保留第二张、绑定第三张并解绑第一张
	updatedContent := "![b](image://2)\n![c](image://3)\n![legacy](https://old.example/legacy.png)"
	if err := service.UpdateArticle(ctx, 1, 100, "新标题", updatedContent, nil, domaincontent.StatusDraft.Int8()); err != nil {
		t.Fatalf("更新图片关系失败: %v", err)
	}
	if imageRepo.images[1].ArticleID != 0 || imageRepo.images[2].ArticleID != 1 || imageRepo.images[3].ArticleID != 1 {
		t.Fatalf("更新后图片关系错误: %+v", imageRepo.images)
	}
	if repo.articles[1].Content != updatedContent {
		t.Fatalf("正文占位符或历史 URL 未原样保存: %s", repo.articles[1].Content)
	}
}

// TestContentServiceRejectsInvalidImageRelations 验证不存在和错误归属图片会回滚文章保存。
func TestContentServiceRejectsInvalidImageRelations(t *testing.T) {
	// 1. 创建引用不存在图片的文章时回滚文章记录
	ctx := context.Background()
	repo := newFakeArticleRepo()
	imageRepo := newFakeArticleImageRepo()
	service := newTestService(repo, imageRepo, &fakeImageStorage{}, nil)
	if err := service.CreateArticle(ctx, 100, "文章", "![missing](image://99)", nil, domaincontent.StatusDraft.Int8()); !errors.Is(err, apperrors.ErrArticleImageInvalid) {
		t.Fatalf("不存在图片错误不正确: %v", err)
	}
	if len(repo.articles) != 0 {
		t.Fatalf("图片校验失败后文章未回滚: %+v", repo.articles)
	}

	// 2. 已绑定其他文章的图片不能用于创建新文章
	image := &domaincontent.ArticleImage{ArticleID: 20, ObjectKey: "article/img/2026/08/owned.png"}
	_ = imageRepo.Create(ctx, image)
	if err := service.CreateArticle(ctx, 100, "文章", "![owned](image://1)", nil, domaincontent.StatusDraft.Int8()); !errors.Is(err, apperrors.ErrArticleImageInvalid) {
		t.Fatalf("错误归属图片错误不正确: %v", err)
	}
	if len(repo.articles) != 0 {
		t.Fatalf("归属冲突后文章未回滚: %+v", repo.articles)
	}
}

// TestContentServiceDetailImageMapping 验证详情只返回正文引用且属于当前文章的图片。
func TestContentServiceDetailImageMapping(t *testing.T) {
	// 1. 构造包含有效、未绑定和缺失图片引用的已发表文章
	ctx := context.Background()
	repo := newFakeArticleRepo()
	imageRepo := newFakeArticleImageRepo()
	imageRepo.images[1] = &domaincontent.ArticleImage{ID: 1, ArticleID: 1, ObjectKey: "article/img/2026/08/a.png"}
	imageRepo.images[2] = &domaincontent.ArticleImage{ID: 2, ObjectKey: "article/img/2026/08/b.png"}
	imageRepo.nextID = 3
	repo.articles[1] = &domaincontent.Article{
		ID: 1, AuthorID: 100, Title: "文章", Content: "![a](image://1) ![again](image://1) ![unbound](image://2) ![missing](image://9)",
		Status: domaincontent.StatusPublished, CreatedTime: time.Now(), UpdatedTime: time.Now(),
	}
	repo.nextID = 2
	service := newTestService(repo, imageRepo, &fakeImageStorage{}, nil)

	// 2. 详情返回原始正文和去重后的有效图片映射
	result, err := service.GetPublishedArticle(ctx, 1, 0)
	if err != nil {
		t.Fatalf("查询文章详情失败: %v", err)
	}
	if result.Content != repo.articles[1].Content || len(result.Images) != 1 || result.Images[0].ID != 1 {
		t.Fatalf("详情图片映射错误: %+v", result)
	}
	if result.Images[0].URL != "https://cdn.example/article/img/2026/08/a.png" {
		t.Fatalf("详情图片 URL 错误: %+v", result.Images)
	}

	// 3. 修改公开域名后正文不变且映射使用新域名
	service.publicDomain = "https://new-cdn.example"
	result, err = service.GetArticle(ctx, 1, 100)
	if err != nil || result.Images[0].URL != "https://new-cdn.example/article/img/2026/08/a.png" {
		t.Fatalf("域名切换后的映射错误: result=%+v err=%v", result, err)
	}
}

// TestContentServiceClearArticleImageCleanup 验证硬删除的图片清理、失败保留和重试语义。
func TestContentServiceClearArticleImageCleanup(t *testing.T) {
	// 1. 创建包含图片的文章并移入垃圾箱
	ctx := context.Background()
	repo := newFakeArticleRepo()
	imageRepo := newFakeArticleImageRepo()
	for _, key := range []string{"a.png", "b.png"} {
		_ = imageRepo.Create(ctx, &domaincontent.ArticleImage{ObjectKey: "article/img/2026/08/" + key})
	}
	storage := &fakeImageStorage{deleteErrors: make(map[string]error)}
	service := newTestService(repo, imageRepo, storage, nil)
	if err := service.CreateArticle(ctx, 100, "文章", "![a](image://1)\n![b](image://2)", nil, domaincontent.StatusDraft.Int8()); err != nil {
		t.Fatalf("创建文章失败: %v", err)
	}
	if err := service.DeleteArticle(ctx, 1, 100); err != nil {
		t.Fatalf("移入垃圾箱失败: %v", err)
	}

	// 2. 任一对象删除失败时保留文章和全部图片记录
	storage.deleteErrors["article/img/2026/08/b.png"] = errors.New("delete image failed")
	if err := service.ClearArticle(ctx, 1, 100); !errors.Is(err, storage.deleteErrors["article/img/2026/08/b.png"]) {
		t.Fatalf("图片清理错误不正确: %v", err)
	}
	if len(repo.articles) != 1 || len(imageRepo.images) != 2 {
		t.Fatalf("图片清理失败后数据库记录不应删除: articles=%+v images=%+v", repo.articles, imageRepo.images)
	}

	// 3. 图片记录删除失败时保留全部数据库记录
	delete(storage.deleteErrors, "article/img/2026/08/b.png")
	imageRepo.deleteErr = errors.New("delete image records failed")
	if err := service.ClearArticle(ctx, 1, 100); !errors.Is(err, imageRepo.deleteErr) {
		t.Fatalf("图片记录删除失败错误不正确: %v", err)
	}
	if len(repo.articles) != 1 || len(imageRepo.images) != 2 {
		t.Fatalf("图片记录删除失败后未回滚: articles=%+v images=%+v", repo.articles, imageRepo.images)
	}

	// 4. 文章记录删除失败时事务回滚图片记录，随后可幂等重试
	imageRepo.deleteErr = nil
	repo.clearErr = errors.New("clear article failed")
	if err := service.ClearArticle(ctx, 1, 100); !errors.Is(err, repo.clearErr) {
		t.Fatalf("数据库删除失败错误不正确: %v", err)
	}
	if len(repo.articles) != 1 || len(imageRepo.images) != 2 {
		t.Fatalf("数据库失败后记录未回滚: articles=%+v images=%+v", repo.articles, imageRepo.images)
	}
	repo.clearErr = nil
	if err := service.ClearArticle(ctx, 1, 100); err != nil {
		t.Fatalf("重试硬删除失败: %v", err)
	}
	if len(repo.articles) != 0 || len(imageRepo.images) != 0 {
		t.Fatalf("硬删除后仍有数据库记录: articles=%+v images=%+v", repo.articles, imageRepo.images)
	}
}

// TestContentServiceUpdateImageConflictRollsBack 验证更新引用错误归属图片时正文保持不变。
func TestContentServiceUpdateImageConflictRollsBack(t *testing.T) {
	// 1. 创建无图片文章并准备属于其他文章的图片
	ctx := context.Background()
	repo := newFakeArticleRepo()
	imageRepo := newFakeArticleImageRepo()
	service := newTestService(repo, imageRepo, &fakeImageStorage{}, nil)
	if err := service.CreateArticle(ctx, 100, "原标题", "原正文", nil, domaincontent.StatusDraft.Int8()); err != nil {
		t.Fatalf("创建文章失败: %v", err)
	}
	_ = imageRepo.Create(ctx, &domaincontent.ArticleImage{ArticleID: 20, ObjectKey: "article/img/2026/08/conflict.png"})

	// 2. 更新引用冲突图片时事务回滚正文和图片关系
	err := service.UpdateArticle(ctx, 1, 100, "新标题", "![conflict](image://1)", nil, domaincontent.StatusDraft.Int8())
	if !errors.Is(err, apperrors.ErrArticleImageInvalid) {
		t.Fatalf("更新图片冲突错误不正确: %v", err)
	}
	if repo.articles[1].Title != "原标题" || repo.articles[1].Content != "原正文" || imageRepo.images[1].ArticleID != 20 {
		t.Fatalf("冲突更新未完整回滚: article=%+v image=%+v", repo.articles[1], imageRepo.images[1])
	}
}

// TestContentServiceDetailWithoutImageReferences 验证无占位符正文返回空图片映射。
func TestContentServiceDetailWithoutImageReferences(t *testing.T) {
	// 1. 创建只有历史完整 URL 的已发表文章
	ctx := context.Background()
	repo := newFakeArticleRepo()
	service := newTestService(repo, newFakeArticleImageRepo(), &fakeImageStorage{}, nil)
	content := "![legacy](https://cdn.example/legacy.png)"
	if err := service.CreateArticle(ctx, 100, "文章", content, nil, domaincontent.StatusPublished.Int8()); err != nil {
		t.Fatalf("创建历史 URL 文章失败: %v", err)
	}

	// 2. 详情保留正文并返回稳定空图片切片
	result, err := service.GetPublishedArticle(ctx, 1, 0)
	if err != nil {
		t.Fatalf("查询文章详情失败: %v", err)
	}
	if result.Content != content || result.Images == nil || len(result.Images) != 0 {
		t.Fatalf("无占位符详情响应错误: %+v", result)
	}
}

// TestContentServiceClearArticleWithoutImages 验证无绑定图片的垃圾箱文章可以直接硬删除。
func TestContentServiceClearArticleWithoutImages(t *testing.T) {
	// 1. 创建无图片文章并移入垃圾箱
	ctx := context.Background()
	repo := newFakeArticleRepo()
	imageRepo := newFakeArticleImageRepo()
	storage := &fakeImageStorage{}
	service := newTestService(repo, imageRepo, storage, nil)
	if err := service.CreateArticle(ctx, 100, "文章", "正文", nil, domaincontent.StatusDraft.Int8()); err != nil {
		t.Fatalf("创建文章失败: %v", err)
	}
	if err := service.DeleteArticle(ctx, 1, 100); err != nil {
		t.Fatalf("移入垃圾箱失败: %v", err)
	}

	// 2. 硬删除不访问 OSS 且清除文章记录
	if err := service.ClearArticle(ctx, 1, 100); err != nil {
		t.Fatalf("硬删除无图片文章失败: %v", err)
	}
	if len(storage.deletedKeys) != 0 || len(repo.articles) != 0 {
		t.Fatalf("无图片硬删除结果错误: deleted=%v articles=%+v", storage.deletedKeys, repo.articles)
	}
}

// cloneArticleImage 复制图片领域对象。
func cloneArticleImage(image *domaincontent.ArticleImage) *domaincontent.ArticleImage {
	// 1. 复制图片值，避免测试调用方直接修改仓储状态
	if image == nil {
		return nil
	}
	copyImage := *image
	return &copyImage
}

// cloneArticleMap 复制文章 Repository 状态。
func cloneArticleMap(articles map[uint64]*domaincontent.Article) map[uint64]*domaincontent.Article {
	// 1. 逐条复制文章聚合
	result := make(map[uint64]*domaincontent.Article, len(articles))
	for id, article := range articles {
		result[id] = cloneArticle(article)
	}
	return result
}

// cloneArticleImageMap 复制图片 Repository 状态。
func cloneArticleImageMap(images map[uint64]*domaincontent.ArticleImage) map[uint64]*domaincontent.ArticleImage {
	// 1. 逐条复制图片记录
	result := make(map[uint64]*domaincontent.ArticleImage, len(images))
	for id, image := range images {
		result[id] = cloneArticleImage(image)
	}
	return result
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
