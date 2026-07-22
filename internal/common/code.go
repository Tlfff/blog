package common

const (
	CodeSuccess = 200
	//------------------------- 请求 ---------------------------------
	CodeBadRequestFormat    = 1000 //请求体JSON格式错误
	CodeInvalidParameter    = 1001 // 参数校验失败
	CodeUnauthorized        = 1002 //未登录/认证失败
	CodeForbidden           = 1003 // 无权限
	CodeDuplicateSubmission = 1004 //重复提交

	CodeInternalServerError = 5000 // 系统异常

	//------------------------- 用户模块 ---------------------------------

	CodeUserExists         = 1100 // 用户已存在
	CodeUserNotFound       = 1101 // 用户不存在
	CodePasswordFailed     = 1102 // 密码错误
	CodeUserDisabled       = 1103 // 用户被禁用
	CodeNickNameNotFound   = 1104 //昵称缺失
	CodePhoneAlreadyExists = 1105 //手机号已被注册
	//------------------------- JWT模块 ---------------------------------

	CodeTokenInvalid = 1200 // Token无效
	CodeTokenExpired = 1201 // Token过期
	//------------------------- 文章模块 ---------------------------------
	CodeArticleNotFound    = 1300 //文章不存在
	CodeArticleDeleted     = 1301 //文章被删除
	CodeArticlePermission  = 1302 //操作文章权限不足
	CodeArticleStatusError = 1303 //文章状态异常
	//------------------------- 评论模块 ---------------------------------
	CodeCommentNotFound    = 1400 // 评论不存在
	CodeCommentDeleted     = 1401 // 评论已被删除
	CodeCommentRootDeleted = 1402 // 主楼评论已被删除，无法回复
	CodeCommentPermission  = 1403 // 操作评论权限不足
	//------------------------- redis模块 ---------------------------------
	CodeUnLockFailed = 1500 // 解锁失败
	CodeLockFailed   = 1501 // 加锁失败
	CodeLockExpired  = 1502 //锁过期
)
