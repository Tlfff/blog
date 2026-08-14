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
	ErrParameter                  = errors.New("参数校验失败")
	ErrForbidden                  = errors.New("权限不足")

	//------------------------- 注册登录模块 ---------------------------------
	ErrRegisterInputEmpty = errors.New("手机号、密码、昵称不能为空")
	ErrLoginInputEmpty    = errors.New("手机号和密码不能为空")
	ErrRoleInvalid        = errors.New("用户角色非法")
	ErrPasswordTooShort   = errors.New("密码长度不能少于6位")
	//------------------------- 用户模块 ---------------------------------
	ErrUserExists          = errors.New("用户已存在")
	ErrUserNotFound        = errors.New("用户不存在或已被禁用")
	ErrPasswordFailed      = errors.New("密码错误")
	ErrPasswordChangeToken = errors.New("密码修改凭证无效或已过期")
	ErrNickNameNotFound    = errors.New("昵称缺失")
	ErrPhoneAlreadyExists  = errors.New("手机号已被注册")
	//------------------------- JWT模块 ---------------------------------
	ErrTokenInvalid   = errors.New("Token无效")
	ErrTokenExpired   = errors.New("Token已过期")
	ErrTokenSignature = errors.New("Token签名错误")
	ErrTokenIssuer    = errors.New("Token签发者错误")
	ErrTokenRevoked   = errors.New("Token已失效")
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
	//------------------------- redis ---------------------------------
	ErrLockFailed   = errors.New("获取redis锁失败")
	ErrLockExpired  = errors.New("redis锁过期")
	ErrUnLockFailed = errors.New("解除redis锁失败,锁不存在或者非该锁持有者")
	//------------------------- 通知模块 ---------------------------------

	//------------------------- kafka模块 ---------------------------------
	ErrKafkaInitFailed       = errors.New("Kafka 初始化失败")
	ErrKafkaBrokerEmpty      = errors.New("Kafka 地址列表为空")
	ErrKafkaSendFailed       = errors.New("Kafka消息发送失败")
	ErrKafkaSerializeFailed  = errors.New("Kafka消息序列化失败")
	ErrKafkaTopicNotConfig   = errors.New("Kafka Topic未配置")
	ErrKafkaConsumeFailed    = errors.New("Kafka消息消费失败")
	ErrKafkaCloseFailed      = errors.New("Kafka关闭失败")
	ErrKafkaNoValidConsumers = errors.New("Kafka 无有效消费者")
	ErrKafkaConsumerRunning  = errors.New("Kafka 消费者已运行")
	ErrKafkaClientClosed     = errors.New("Kafka 客户端已关闭")
	ErrKafkaPingFailed       = errors.New("Kafka 预热连接失败")
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
	case ErrParameter,
		ErrRegisterInputEmpty,
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
	case ErrPasswordChangeToken:
		return CodeUnauthorized
	case ErrNickNameNotFound:
		return CodeNickNameNotFound
	case ErrPhoneAlreadyExists:
		return CodePhoneAlreadyExists

	// JWT
	case ErrTokenInvalid,
		ErrTokenSignature,
		ErrTokenIssuer,
		ErrTokenRevoked:
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

	// redis
	case ErrLockExpired:
		return CodeLockExpired
	case ErrLockFailed:
		return CodeLockFailed
	case ErrUnLockFailed:
		return CodeUnLockFailed
	// kafka
	case ErrKafkaInitFailed,
		ErrKafkaBrokerEmpty:
		return CodeKafkaInitFailed
	case ErrKafkaSendFailed:
		return CodeKafkaSendFailed
	case ErrKafkaSerializeFailed:
		return CodeKafkaSerializeFailed
	case ErrKafkaTopicNotConfig:
		return CodeKafkaTopicNotConfig
	case ErrKafkaConsumeFailed:
		return CodeKafkaConsumeFailed
	case ErrKafkaCloseFailed:
		return CodeKafkaCloseFailed
	case ErrKafkaNoValidConsumers:
		return CodeKafkaNoValidConsumers
	case ErrKafkaConsumerRunning:
		return CodeKafkaConsumerRunning
	case ErrKafkaClientClosed:
		return CodeKafkaClientClosed
	case ErrKafkaPingFailed:
		return CodeKafkaPingFailed
	default:
		return CodeInternalServerError

	}
}
