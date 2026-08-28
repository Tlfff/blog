package response

// 统一业务响应码：200 表示成功，其余按模块划分号段
const (
	CodeSuccess = 200 // 请求成功
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
	//------------------------- kafka模块 ---------------------------------
	CodeKafkaInitFailed       = 1600 // Kafka初始化失败
	CodeKafkaSendFailed       = 1601 // Kafka消息发送失败
	CodeKafkaSerializeFailed  = 1602 // Kafka消息序列化失败
	CodeKafkaTopicNotConfig   = 1603 // Kafka Topic未配置
	CodeKafkaConsumeFailed    = 1604 // Kafka消息消费失败
	CodeKafkaCloseFailed      = 1605 // Kafka关闭失败
	CodeKafkaNoValidConsumers = 1606 // Kafka 无有效消费者
	CodeKafkaConsumerRunning  = 1607 // Kafka 消费者已运行
	CodeKafkaClientClosed     = 1608 // Kafka 客户端已关闭
	CodeKafkaPingFailed       = 1609 // Kafka 预热连接失败
	//------------------------- 搜索模块 ---------------------------------
	CodeSearchUnavailable = 1700 // 文章搜索服务暂不可用
)
