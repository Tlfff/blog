package domain

import "errors"

// Article 领域错误，供 Application 层判断并映射为对外错误码
var (
	ErrArticleNotFound         = errors.New("文章不存在")   // 按ID查询文章时未命中
	ErrArticleDeleted          = errors.New("文章已删除")   // 目标文章处于已删除状态，不允许继续操作
	ErrArticlePermissionDenied = errors.New("无权操作该文章") // 操作者不是文章作者且非管理员
	ErrArticleStatusError      = errors.New("文章状态异常")  // 当前状态不允许执行该状态流转
)
