package response

import apperrors "blog/internal/shared/apperrors"

// 把统一业务错误映射为对外业务响应码，未登记的错误统一按系统异常处理
func CodeByError(err error) int {

	// 1. 按错误值逐类匹配，命中则返回对应业务码
	switch err {
	// 请求
	case apperrors.ErrInvalidRequestBody:
		return CodeBadRequestFormat
	case apperrors.ErrAuthorizationRequired,
		apperrors.ErrInvalidAuthorizationHeader,
		apperrors.ErrTokenEmpty:

		return CodeUnauthorized
	case apperrors.ErrDuplicateSubmission:
		return CodeDuplicateSubmission
	case apperrors.ErrForbidden:
		return CodeForbidden

	// 参数错误
	case apperrors.ErrParameter,
		apperrors.ErrRegisterInputEmpty,
		apperrors.ErrLoginInputEmpty,
		apperrors.ErrRoleInvalid,
		apperrors.ErrPasswordTooShort,
		apperrors.ErrArticleTitleEmpty,
		apperrors.ErrArticleContentEmpty,
		apperrors.ErrArticleIDInvalid,
		apperrors.ErrArticleStatusInvalid:

		return CodeInvalidParameter

	// 用户模块
	case apperrors.ErrUserExists:
		return CodeUserExists
	case apperrors.ErrUserNotFound:
		return CodeUserNotFound
	case apperrors.ErrPasswordFailed:
		return CodePasswordFailed
	case apperrors.ErrPasswordChangeToken:
		return CodeUnauthorized
	case apperrors.ErrNickNameNotFound:
		return CodeNickNameNotFound
	case apperrors.ErrPhoneAlreadyExists:
		return CodePhoneAlreadyExists

	// JWT
	case apperrors.ErrTokenInvalid,
		apperrors.ErrTokenSignature,
		apperrors.ErrTokenIssuer,
		apperrors.ErrTokenRevoked:
		return CodeTokenInvalid
	case apperrors.ErrTokenExpired:
		return CodeTokenExpired
	// 文章模块
	case apperrors.ErrArticleNotFound:
		return CodeArticleNotFound
	case apperrors.ErrArticleDeleted:
		return CodeArticleDeleted
	case apperrors.ErrArticlePermissionDenied:
		return CodeArticlePermission
	case apperrors.ErrArticleStatusError:
		return CodeArticleStatusError
	// 评论模块
	case apperrors.ErrCommentNotFound:
		return CodeCommentNotFound
	case apperrors.ErrCommentDeleted:
		return CodeCommentDeleted
	case apperrors.ErrCommentRootDeleted:
		return CodeCommentRootDeleted
	case apperrors.ErrCommentPermission:
		return CodeCommentPermission

	// redis
	case apperrors.ErrLockExpired:
		return CodeLockExpired
	case apperrors.ErrLockFailed:
		return CodeLockFailed
	case apperrors.ErrUnLockFailed:
		return CodeUnLockFailed
	// kafka
	case apperrors.ErrKafkaInitFailed,
		apperrors.ErrKafkaBrokerEmpty:
		return CodeKafkaInitFailed
	case apperrors.ErrKafkaSendFailed:
		return CodeKafkaSendFailed
	case apperrors.ErrKafkaSerializeFailed:
		return CodeKafkaSerializeFailed
	case apperrors.ErrKafkaTopicNotConfig:
		return CodeKafkaTopicNotConfig
	case apperrors.ErrKafkaConsumeFailed:
		return CodeKafkaConsumeFailed
	case apperrors.ErrKafkaCloseFailed:
		return CodeKafkaCloseFailed
	case apperrors.ErrKafkaNoValidConsumers:
		return CodeKafkaNoValidConsumers
	case apperrors.ErrKafkaConsumerRunning:
		return CodeKafkaConsumerRunning
	case apperrors.ErrKafkaClientClosed:
		return CodeKafkaClientClosed
	case apperrors.ErrKafkaPingFailed:
		return CodeKafkaPingFailed
	// 2. 未登记的错误统一返回系统异常码，避免泄漏内部细节
	default:
		return CodeInternalServerError

	}
}
