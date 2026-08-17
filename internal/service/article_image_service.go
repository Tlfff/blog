package service

import (
	"blog/internal/common"
	"blog/pkg/oss"
	"context"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
)

// 负责文章内嵌图片的直传凭证生成
// 图片先上传到 temp 目录，等文章保存/发布时再转正到正式目录
type ArticleImageService struct {
	oss             *oss.MinioClient
	ossPublicDomain string
	allowedExts     map[string]bool // 允许上传的图片扩展名
}

// 注入 MinIO 客户端、公开域名和允许的扩展名
func NewArticleImageService(ossClient *oss.MinioClient, publicDomain string, allowedExts []string) *ArticleImageService {
	extMap := make(map[string]bool, len(allowedExts))
	for _, ext := range allowedExts {
		extMap[strings.ToLower(ext)] = true
	}
	return &ArticleImageService{
		oss:             ossClient,
		ossPublicDomain: publicDomain,
		allowedExts:     extMap,
	}
}

// 获取文章图片上传凭证
// 生成 objectKey: article/temp/{uuid}.{ext}，保存/发布时再按文章ID转正到正式目录
func (s *ArticleImageService) GetUploadURL(ctx context.Context, fileExt string) (uploadURL, url string, err error) {
	if s.oss == nil {
		return "", "", common.ErrSystem
	}

	// 校验文件扩展名白名单（来自配置）
	ext := strings.ToLower(strings.TrimPrefix(fileExt, "."))
	if !s.allowedExts[ext] {
		return "", "", common.ErrInvalidRequestBody
	}

	objectKey := path.Join("article", "temp", uuid.NewString()+"."+ext)

	// 生成预签名 PUT URL，有效期 10 分钟
	uploadURL, err = s.oss.PresignedPutURL(ctx, objectKey, 10*time.Minute)
	if err != nil {
		return "", "", err
	}

	// 直接返回完整访问 URL，前端上传成功后插入 Markdown 即可
	url = s.oss.GetObjectURL(s.ossPublicDomain, objectKey)
	return uploadURL, url, nil
}
