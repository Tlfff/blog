package infra

import (
	domaincontent "blog/internal/article/domain"
	"blog/internal/article/infra/model"
	platformtransaction "blog/internal/platform/transaction"
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// articleImageRepository 是文章图片 Repository 的 GORM 实现。
type articleImageRepository struct {
	db *gorm.DB // GORM 数据库连接
}

// NewArticleImageRepository 创建文章图片 Repository。
func NewArticleImageRepository(db *gorm.DB) domaincontent.ArticleImageRepository {
	// 1. 返回持有共享数据库连接的 Repository
	return &articleImageRepository{db: db}
}

// Create 创建未绑定文章的图片记录并回填数据库字段。
func (r *articleImageRepository) Create(ctx context.Context, image *domaincontent.ArticleImage) error {
	// 1. 使用事务上下文中的连接写入图片记录
	imageModel := toModelArticleImage(image)
	if err := platformtransaction.DB(ctx, r.db).WithContext(ctx).Create(imageModel).Error; err != nil {
		return err
	}

	// 2. 回填数据库生成的唯一标识和创建时间
	image.ID = imageModel.ID
	image.CreatedTime = imageModel.CreatedTime
	return nil
}

// FindByIDs 按图片唯一标识集合批量查询图片。
func (r *articleImageRepository) FindByIDs(ctx context.Context, ids []uint64) ([]*domaincontent.ArticleImage, error) {
	// 1. 空集合不访问数据库
	if len(ids) == 0 {
		return []*domaincontent.ArticleImage{}, nil
	}

	// 2. 批量查询并转换为领域对象
	var imageModels []*model.ArticleImage
	if err := platformtransaction.DB(ctx, r.db).WithContext(ctx).
		Where("id IN ?", ids).
		Find(&imageModels).Error; err != nil {
		return nil, err
	}
	return toDomainArticleImages(imageModels), nil
}

// FindByIDsForUpdate 在当前事务中锁定并查询图片记录。
func (r *articleImageRepository) FindByIDsForUpdate(ctx context.Context, ids []uint64) ([]*domaincontent.ArticleImage, error) {
	// 1. 空集合不访问数据库
	if len(ids) == 0 {
		return []*domaincontent.ArticleImage{}, nil
	}

	// 2. 使用行锁读取图片，避免并发绑定覆盖关系
	var imageModels []*model.ArticleImage
	if err := platformtransaction.DB(ctx, r.db).WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id IN ?", ids).
		Find(&imageModels).Error; err != nil {
		return nil, err
	}
	return toDomainArticleImages(imageModels), nil
}

// FindByArticleIDAndIDs 查询正文引用且属于指定文章的图片。
func (r *articleImageRepository) FindByArticleIDAndIDs(ctx context.Context, articleID uint64, ids []uint64) ([]*domaincontent.ArticleImage, error) {
	// 1. 空集合不访问数据库
	if len(ids) == 0 {
		return []*domaincontent.ArticleImage{}, nil
	}

	// 2. 同时按文章归属和图片标识过滤
	var imageModels []*model.ArticleImage
	if err := platformtransaction.DB(ctx, r.db).WithContext(ctx).
		Where("article_id = ? AND id IN ?", articleID, ids).
		Find(&imageModels).Error; err != nil {
		return nil, err
	}
	return toDomainArticleImages(imageModels), nil
}

// FindByArticleID 查询指定文章已绑定的全部图片。
func (r *articleImageRepository) FindByArticleID(ctx context.Context, articleID uint64) ([]*domaincontent.ArticleImage, error) {
	// 1. 按文章归属批量查询图片
	var imageModels []*model.ArticleImage
	if err := platformtransaction.DB(ctx, r.db).WithContext(ctx).
		Where("article_id = ?", articleID).
		Find(&imageModels).Error; err != nil {
		return nil, err
	}
	return toDomainArticleImages(imageModels), nil
}

// BindArticle 批量绑定尚未归属文章的图片并返回影响行数。
func (r *articleImageRepository) BindArticle(ctx context.Context, ids []uint64, articleID uint64) (int64, error) {
	// 1. 空集合视为无需更新
	if len(ids) == 0 {
		return 0, nil
	}

	// 2. 只绑定 article_id 为空的图片，避免覆盖已有归属
	result := platformtransaction.DB(ctx, r.db).WithContext(ctx).
		Model(&model.ArticleImage{}).
		Where("id IN ? AND article_id IS NULL", ids).
		Update("article_id", articleID)
	return result.RowsAffected, result.Error
}

// UnbindArticle 批量解绑属于指定文章的图片并返回影响行数。
func (r *articleImageRepository) UnbindArticle(ctx context.Context, ids []uint64, articleID uint64) (int64, error) {
	// 1. 空集合视为无需更新
	if len(ids) == 0 {
		return 0, nil
	}

	// 2. 只解绑当前文章拥有的图片，避免修改其他文章关系
	result := platformtransaction.DB(ctx, r.db).WithContext(ctx).
		Model(&model.ArticleImage{}).
		Where("id IN ? AND article_id = ?", ids, articleID).
		Update("article_id", nil)
	return result.RowsAffected, result.Error
}

// DeleteByArticleID 删除指定文章的全部图片记录并返回影响行数。
func (r *articleImageRepository) DeleteByArticleID(ctx context.Context, articleID uint64) (int64, error) {
	// 1. 按文章归属删除图片记录
	result := platformtransaction.DB(ctx, r.db).WithContext(ctx).
		Where("article_id = ?", articleID).
		Delete(&model.ArticleImage{})
	return result.RowsAffected, result.Error
}

// toDomainArticleImage 将图片持久化模型转换为领域对象。
func toDomainArticleImage(imageModel *model.ArticleImage) *domaincontent.ArticleImage {
	// 1. 将可空文章标识转换为领域零值语义
	articleID := uint64(0)
	if imageModel.ArticleID != nil {
		articleID = *imageModel.ArticleID
	}
	return &domaincontent.ArticleImage{
		ID: imageModel.ID, ArticleID: articleID, ObjectKey: imageModel.ObjectKey, CreatedTime: imageModel.CreatedTime,
	}
}

// toDomainArticleImages 批量转换图片持久化模型。
func toDomainArticleImages(imageModels []*model.ArticleImage) []*domaincontent.ArticleImage {
	// 1. 按查询结果顺序转换图片模型
	images := make([]*domaincontent.ArticleImage, 0, len(imageModels))
	for _, imageModel := range imageModels {
		images = append(images, toDomainArticleImage(imageModel))
	}
	return images
}

// toModelArticleImage 将图片领域对象转换为持久化模型。
func toModelArticleImage(image *domaincontent.ArticleImage) *model.ArticleImage {
	// 1. 仅在图片已绑定文章时写入非空文章标识
	var articleID *uint64
	if image.ArticleID > 0 {
		value := image.ArticleID
		articleID = &value
	}
	return &model.ArticleImage{
		ID: image.ID, ArticleID: articleID, ObjectKey: image.ObjectKey, CreatedTime: image.CreatedTime,
	}
}
