package common

import (
	"errors"
)

var (
	//------------------------- 系统 ---------------------------------
	ErrPasswordHashFailed = errors.New("密码加密失败")
	ErrSystem             = errors.New("系统异常")
	//------------------------- 请求 ---------------------------------
	ErrInvalidRequestBody         = errors.New("请求体格式错误")
	ErrAuthorizationRequired      = errors.New("未携带登录凭证")
	ErrInvalidAuthorizationHeader = errors.New("Authorization格式错误")
	ErrTokenEmpty                 = errors.New("Token不能为空")
	ErrDuplicateSubmission        = errors.New("请勿重复提交请求")
	ErrForbidden                  = errors.New("权限不足")

	//------------------------- 注册登录模块 ---------------------------------
	ErrRegisterInputEmpty = errors.New("手机号、密码、昵称不能为空")
	ErrLoginInputEmpty    = errors.New("手机号和密码不能为空")
	ErrRoleInvalid        = errors.New("用户角色非法")
	ErrPasswordTooShort   = errors.New("密码长度不能少于6位")
	//------------------------- 用户模块 ---------------------------------
	ErrUserExists         = errors.New("用户已存在")
	ErrUserNotFound       = errors.New("用户不存在或已被禁用")
	ErrPasswordFailed     = errors.New("密码错误")
	ErrNickNameNotFound   = errors.New("昵称缺失")
	ErrPhoneAlreadyExists = errors.New("手机号已被注册")
	//------------------------- JWT模块 ---------------------------------
	ErrTokenInvalid   = errors.New("Token无效")
	ErrTokenExpired   = errors.New("Token已过期")
	ErrTokenSignature = errors.New("Token签名错误")
	ErrTokenIssuer    = errors.New("Token签发者错误")
	//------------------------- 文章模块 ---------------------------------
	ErrArticleNotFound         = errors.New("文章不存在")
	ErrArticleDeleted          = errors.New("文章已删除")
	ErrArticlePermissionDenied = errors.New("无权操作该文章")
	ErrArticleStatusError      = errors.New("文章状态异常")
	ErrArticleTitleEmpty       = errors.New("文章标题不能为空")
	ErrArticleContentEmpty     = errors.New("文章内容不能为空")
	ErrArticleIDInvalid        = errors.New("文章ID非法")
	ErrArticleStatusInvalid    = errors.New("文章状态非法")
	//------------------------- 评论模块 ---------------------------------
	ErrCommentNotFound    = errors.New("评论不存在")
	ErrCommentDeleted     = errors.New("评论已被删除")
	ErrCommentRootDeleted = errors.New("主楼评论已被删除，无法回复")
	ErrCommentPermission  = errors.New("无权操作该评论")
)

func GetCodeByError(err error) int {

	switch err {
	// 请求
	case ErrInvalidRequestBody:
		return CodeBadRequestFormat
	case ErrAuthorizationRequired,
		ErrInvalidAuthorizationHeader,
		ErrTokenEmpty:

		return CodeUnauthorized
	case ErrDuplicateSubmission:
		return CodeDuplicateSubmission
	case ErrForbidden:
		return CodeForbidden

	// 参数错误
	case ErrRegisterInputEmpty,
		ErrLoginInputEmpty,
		ErrRoleInvalid,
		ErrPasswordTooShort,
		ErrArticleTitleEmpty,
		ErrArticleContentEmpty,
		ErrArticleIDInvalid,
		ErrArticleStatusInvalid:

		return CodeInvalidParameter

	// 用户模块
	case ErrUserExists:
		return CodeUserExists
	case ErrUserNotFound:
		return CodeUserNotFound
	case ErrPasswordFailed:
		return CodePasswordFailed
	case ErrNickNameNotFound:
		return CodeNickNameNotFound
	case ErrPhoneAlreadyExists:
		return CodePhoneAlreadyExists

	// JWT
	case ErrTokenInvalid,
		ErrTokenSignature,
		ErrTokenIssuer:
		return CodeTokenInvalid
	case ErrTokenExpired:
		return CodeTokenExpired
	// 文章模块
	case ErrArticleNotFound:
		return CodeArticleNotFound
	case ErrArticleDeleted:
		return CodeArticleDeleted
	case ErrArticlePermissionDenied:
		return CodeArticlePermission
	case ErrArticleStatusError:
		return CodeArticleStatusError
	// 评论模块
	case ErrCommentNotFound:
		return CodeCommentNotFound
	case ErrCommentDeleted:
		return CodeCommentDeleted
	case ErrCommentRootDeleted:
		return CodeCommentRootDeleted
	case ErrCommentPermission:
		return CodeCommentPermission
	default:
		return CodeInternalServerError

	}
}
