package app

import (
	articledto "blog/internal/article/app/dto"
	domaincontent "blog/internal/article/domain"
	apperrors "blog/internal/shared/apperrors"
	"context"
	"errors"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
)

const articleImageUploadURLTTL = 10 * time.Minute

// GetImageUploadURL 生成单张文章图片上传凭证并创建未绑定图片记录。
func (s *Service) GetImageUploadURL(ctx context.Context, command GetImageUploadURLCommand) (*articledto.ImageUploadCredentialResponse, error) {
	// 1. 校验依赖和图片扩展名
	if s.imageStorage == nil || s.imageRepo == nil {
		return nil, apperrors.ErrSystem
	}
	extension := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(command.FileExt), "."))
	if !s.allowedExts[extension] {
		return nil, apperrors.ErrInvalidRequestBody
	}

	// 2. 生成独立对象 Key 和预签名上传地址
	now := s.now()
	objectKey := path.Join("article", "img", now.Format("2006"), now.Format("01"), uuid.NewString()+"."+extension)
	uploadURL, err := s.imageStorage.PresignedPutURL(ctx, objectKey, articleImageUploadURLTTL)
	if err != nil {
		return nil, err
	}

	// 3. 创建未绑定图片记录并返回实时预览信息
	image := &domaincontent.ArticleImage{ObjectKey: objectKey}
	if err := s.imageRepo.Create(ctx, image); err != nil {
		return nil, err
	}
	return &articledto.ImageUploadCredentialResponse{
		ImageID: image.ID, UploadURL: uploadURL, URL: s.imageStorage.GetObjectURL(s.publicDomain, objectKey),
	}, nil
}

// extractImageIDs 提取正文中的系统图片 ID。
func (s *Service) extractImageIDs(content string) ([]uint64, error) {
	// 1. 正文解析器是图片关系同步的必要依赖
	if s.imageReferences == nil {
		return nil, apperrors.ErrSystem
	}
	return s.imageReferences.Extract(content)
}

// bindArticleImages 校验正文图片归属并绑定未归属图片。
func (s *Service) bindArticleImages(ctx context.Context, articleID uint64, imageIDs []uint64, allowCurrentArticle bool) error {
	// 1. 无图片引用时无需访问图片 Repository
	if len(imageIDs) == 0 {
		return nil
	}
	if s.imageRepo == nil {
		return apperrors.ErrSystem
	}

	// 2. 锁定全部图片并验证记录完整性和当前归属
	images, err := s.imageRepo.FindByIDsForUpdate(ctx, imageIDs)
	if err != nil {
		return err
	}
	if len(images) != len(imageIDs) {
		return apperrors.ErrArticleImageInvalid
	}
	imagesByID := make(map[uint64]*domaincontent.ArticleImage, len(images))
	for _, image := range images {
		imagesByID[image.ID] = image
	}
	unboundIDs := make([]uint64, 0, len(imageIDs))
	for _, imageID := range imageIDs {
		image := imagesByID[imageID]
		if image == nil {
			return apperrors.ErrArticleImageInvalid
		}
		switch {
		case image.ArticleID == 0:
			unboundIDs = append(unboundIDs, imageID)
		case allowCurrentArticle && image.ArticleID == articleID:
			continue
		default:
			return apperrors.ErrArticleImageInvalid
		}
	}

	// 3. 使用条件更新绑定未归属图片，并校验影响行数
	rows, err := s.imageRepo.BindArticle(ctx, unboundIDs, articleID)
	if err != nil {
		return err
	}
	if rows != int64(len(unboundIDs)) {
		return apperrors.ErrArticleImageInvalid
	}
	return nil
}

// unbindArticleImages 将正文已移除且仍属于当前文章的图片解除绑定。
func (s *Service) unbindArticleImages(ctx context.Context, articleID uint64, imageIDs []uint64) error {
	// 1. 无移除图片时无需访问图片 Repository
	if len(imageIDs) == 0 {
		return nil
	}
	if s.imageRepo == nil {
		return apperrors.ErrSystem
	}

	// 2. 锁定候选图片并筛选当前文章实际拥有的记录
	images, err := s.imageRepo.FindByIDsForUpdate(ctx, imageIDs)
	if err != nil {
		return err
	}
	boundIDs := make([]uint64, 0, len(images))
	for _, image := range images {
		if image.ArticleID == articleID {
			boundIDs = append(boundIDs, image.ID)
		}
	}

	// 3. 条件解绑并校验关系没有被并发修改
	rows, err := s.imageRepo.UnbindArticle(ctx, boundIDs, articleID)
	if err != nil {
		return err
	}
	if rows != int64(len(boundIDs)) {
		return apperrors.ErrArticleImageInvalid
	}
	return nil
}

// buildArticleImageResponses 构造正文引用且属于当前文章的图片映射。
func (s *Service) buildArticleImageResponses(ctx context.Context, articleID uint64, content string) ([]articledto.ArticleImageResponse, error) {
	// 1. 提取正文图片 ID，无引用时返回稳定空切片
	imageIDs, err := s.extractImageIDs(content)
	if err != nil {
		return nil, err
	}
	if len(imageIDs) == 0 {
		return []articledto.ArticleImageResponse{}, nil
	}
	if s.imageRepo == nil || s.imageStorage == nil {
		return nil, apperrors.ErrSystem
	}

	// 2. 批量查询当前文章图片，并按正文首次出现顺序生成映射
	images, err := s.imageRepo.FindByArticleIDAndIDs(ctx, articleID, imageIDs)
	if err != nil {
		return nil, err
	}
	imagesByID := make(map[uint64]*domaincontent.ArticleImage, len(images))
	for _, image := range images {
		imagesByID[image.ID] = image
	}
	responses := make([]articledto.ArticleImageResponse, 0, len(images))
	for _, imageID := range imageIDs {
		image, ok := imagesByID[imageID]
		if !ok {
			continue
		}
		responses = append(responses, articledto.ArticleImageResponse{
			ID: image.ID, URL: s.imageStorage.GetObjectURL(s.publicDomain, image.ObjectKey),
		})
	}
	return responses, nil
}

// differenceImageIDs 返回左侧集合中未出现在右侧集合的图片 ID。
func differenceImageIDs(left, right []uint64) []uint64 {
	// 1. 建立右侧集合后保留左侧差集，并维持原顺序
	rightSet := make(map[uint64]struct{}, len(right))
	for _, imageID := range right {
		rightSet[imageID] = struct{}{}
	}
	difference := make([]uint64, 0)
	for _, imageID := range left {
		if _, exists := rightSet[imageID]; !exists {
			difference = append(difference, imageID)
		}
	}
	return difference
}

// normalizeArticleTransactionError 还原需要对外识别的事务内业务错误。
func normalizeArticleTransactionError(err error) error {
	// 1. 事务协调器会包装原始错误，对外业务错误需要恢复为稳定错误值
	if errors.Is(err, apperrors.ErrArticleImageInvalid) {
		return apperrors.ErrArticleImageInvalid
	}
	return err
}
