package content

import "errors"

var (
	ErrArticleNotFound        = errors.New("文章不存在")
	ErrArticleDeleted         = errors.New("文章已删除")
	ErrArticlePermissionDenied = errors.New("无权操作该文章")
	ErrArticleStatusError     = errors.New("文章状态异常")
)
