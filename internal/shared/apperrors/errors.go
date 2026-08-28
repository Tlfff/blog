package apperrors

import (
	"errors"
)

// 统一业务错误，供 Handler 通过 c.Error 抛出，由错误中间件映射为响应码
var (
	//------------------------- 系统 ---------------------------------
	ErrPasswordHashFailed = errors.New("密码加密失败") // 密码哈希计算失败
	ErrSystem             = errors.New("系统异常")   // 未归类的系统内部异常
	//------------------------- 请求 ---------------------------------
	ErrInvalidRequestBody         = errors.New("请求体格式错误")           // 请求体JSON无法解析
	ErrAuthorizationRequired      = errors.New("未携带登录凭证")           // 请求未携带 Authorization 头
	ErrInvalidAuthorizationHeader = errors.New("Authorization格式错误") // Authorization 头格式不符合 Bearer 规范
	ErrTokenEmpty                 = errors.New("Token不能为空")         // Token 为空字符串
	ErrDuplicateSubmission        = errors.New("请勿重复提交请求")          // 命中重复提交拦截
	ErrParameter                  = errors.New("参数校验失败")            // binding 参数校验未通过
	ErrForbidden                  = errors.New("权限不足")              // 已登录但权限不足

	//------------------------- 注册登录模块 ---------------------------------
	ErrRegisterInputEmpty = errors.New("手机号、密码、昵称不能为空") // 注册时必填项为空
	ErrLoginInputEmpty    = errors.New("手机号和密码不能为空")    // 登录时必填项为空
	ErrRoleInvalid        = errors.New("用户角色非法")        // 角色取值不在允许范围内
	ErrPasswordTooShort   = errors.New("密码长度不能少于6位")    // 密码长度不足最小限制
	//------------------------- 用户模块 ---------------------------------
	ErrUserExists          = errors.New("用户已存在")        // 注册时账号已存在
	ErrUserNotFound        = errors.New("用户不存在或已被禁用")   // 用户不存在或状态为禁用
	ErrPasswordFailed      = errors.New("密码错误")         // 登录或校验旧密码时密码不匹配
	ErrPasswordChangeToken = errors.New("密码修改凭证无效或已过期") // 一次性改密凭证无效或已过期
	ErrNickNameNotFound    = errors.New("昵称缺失")         // 登录参数中缺少昵称
	ErrPhoneAlreadyExists  = errors.New("手机号已被注册")      // 注册或换绑时手机号唯一性冲突
	//------------------------- JWT模块 ---------------------------------
	ErrTokenInvalid   = errors.New("Token无效")    // Token 无法解析或内容非法
	ErrTokenExpired   = errors.New("Token已过期")   // Token 已超过有效期
	ErrTokenSignature = errors.New("Token签名错误")  // Token 签名校验失败
	ErrTokenIssuer    = errors.New("Token签发者错误") // Token 签发者与预期不一致
	ErrTokenRevoked   = errors.New("Token已失效")   // Token 已被登出或强制下线
	//------------------------- 文章模块 ---------------------------------
	ErrArticleNotFound         = errors.New("文章不存在")    // 按ID查询文章未命中
	ErrArticleDeleted          = errors.New("文章已删除")    // 文章处于已删除状态
	ErrArticlePermissionDenied = errors.New("无权操作该文章")  // 操作者不是文章作者
	ErrArticleStatusError      = errors.New("文章状态异常")   // 文章当前状态不允许该操作
	ErrArticleTitleEmpty       = errors.New("文章标题不能为空") // 创建或更新时标题为空
	ErrArticleContentEmpty     = errors.New("文章内容不能为空") // 创建或更新时正文为空
	ErrArticleIDInvalid        = errors.New("文章ID非法")   // 文章ID为0或非法值
	ErrArticleStatusInvalid    = errors.New("文章状态非法")   // 文章状态取值不在允许范围内
	//------------------------- 搜索模块 ---------------------------------
	ErrSearchUnavailable = errors.New("文章搜索服务暂不可用") // Elasticsearch 查询服务不可用
	//------------------------- 评论模块 ---------------------------------
	ErrCommentNotFound    = errors.New("评论不存在")         // 按ID查询评论未命中
	ErrCommentDeleted     = errors.New("评论已被删除")        // 评论处于已删除状态
	ErrCommentRootDeleted = errors.New("主楼评论已被删除，无法回复") // 回复时主楼评论已被删除
	ErrCommentPermission  = errors.New("无权操作该评论")       // 操作者不是评论作者且非管理员
	//------------------------- redis ---------------------------------
	ErrLockFailed   = errors.New("获取redis锁失败")              // 重试后仍未取得分布式锁
	ErrLockExpired  = errors.New("redis锁过期")                // 持锁期间锁已过期
	ErrUnLockFailed = errors.New("解除redis锁失败,锁不存在或者非该锁持有者") // 解锁时锁不存在或持有者不匹配
	//------------------------- 通知模块 ---------------------------------

	//------------------------- kafka模块 ---------------------------------
	ErrKafkaInitFailed       = errors.New("Kafka 初始化失败")    // Kafka 客户端初始化失败
	ErrKafkaBrokerEmpty      = errors.New("Kafka 地址列表为空")   // 配置中 Kafka broker 列表为空
	ErrKafkaSendFailed       = errors.New("Kafka消息发送失败")    // 消息发送到 Kafka 失败
	ErrKafkaSerializeFailed  = errors.New("Kafka消息序列化失败")   // 消息序列化为JSON失败
	ErrKafkaTopicNotConfig   = errors.New("Kafka Topic未配置") // 对应业务的 Topic 未在配置中声明
	ErrKafkaConsumeFailed    = errors.New("Kafka消息消费失败")    // 消费消息过程中出错
	ErrKafkaCloseFailed      = errors.New("Kafka关闭失败")      // 关闭 Kafka 连接失败
	ErrKafkaNoValidConsumers = errors.New("Kafka 无有效消费者")   // 启动消费时没有任何有效消费者
	ErrKafkaConsumerRunning  = errors.New("Kafka 消费者已运行")   // 重复启动同一消费者
	ErrKafkaClientClosed     = errors.New("Kafka 客户端已关闭")   // 在已关闭的客户端上继续操作
	ErrKafkaPingFailed       = errors.New("Kafka 预热连接失败")   // 启动时预热 Kafka 连接失败
)
