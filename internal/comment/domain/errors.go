package domain

import "errors"

var (
	ErrCommentNotFound    = errors.New("评论不存在")
	ErrCommentDeleted     = errors.New("评论已被删除")
	ErrCommentRootDeleted = errors.New("主楼评论已被删除，无法回复")
	ErrCommentPermission  = errors.New("无权操作该评论")
)
