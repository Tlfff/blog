package community

import "errors"

// Community 领域错误，供 Application 层判断并映射为对外错误码
var (
	ErrCommentNotFound    = errors.New("评论不存在")         // 按ID查询评论时未命中
	ErrCommentDeleted     = errors.New("评论已被删除")        // 操作目标评论已处于删除状态
	ErrCommentRootDeleted = errors.New("主楼评论已被删除，无法回复") // 回复时主楼评论已被删除
	ErrCommentPermission  = errors.New("无权操作该评论")       // 非评论作者且非管理员尝试删除评论
)
