# 阶段一行为对比

本文档对比 DDD 四层重构前后的关键外部行为，作为阶段一验收依据。所有结论均以契约测试、应用层测试与全量测试验证为准。

## 1. HTTP 接口对比

| 维度 | 重构前 | 重构后 | 结论 |
| --- | --- | --- | --- |
| 路由方法/路径 | `internal/routes` 注册 | `internal/interfaces/http/routes` 注册 | 一致 |
| 鉴权分组 | public/optional/private/admin | 相同，中间件迁移到 Interfaces | 一致 |
| 统一响应结构 | `{success,code,message,data}` | 相同 | 一致 |
| HTTP 状态码 | 业务接口固定 200 | 相同 | 一致 |
| 业务错误码 | `internal/common` 常量 | 相同，Application 映射到原错误 | 一致 |

契约测试：[contract_test.go](/Users/staff/code/go/blog-system/blog/internal/interfaces/http/routes/contract_test.go)、[contract_test.go](/Users/staff/code/go/blog-system/blog/internal/common/contract_test.go)。

## 2. 开放 gRPC 对比

| Service/Method | 重构前 | 重构后 | 结论 |
| --- | --- | --- | --- |
| UserService.GetUserBasicInfo | service.UserService | Identity Application | 一致 |
| UserService.GetPublicUserInfo | service.UserService | Identity Application | 一致 |
| ArticleService.GetAvailableList | service.ArticleService | Content Application | 一致 |
| CommentService.GetCommentStats | service.CommentService | Community Application | 一致 |
| gRPC 错误映射 | NotFound/PermissionDenied/Unauthenticated/InvalidArgument/Internal | 相同 | 一致 |
| 鉴权拦截 | JWT/HMAC/Trace | 相同，迁移到 Interfaces | 一致 |

契约测试：[contract_test.go](/Users/staff/code/go/blog-system/blog/internal/interfaces/grpc/server/contract_test.go)、[error_test.go](/Users/staff/code/go/blog-system/blog/internal/interfaces/grpc/handler/error_test.go)。

## 3. 权限结果对比

| 场景 | 重构前 | 重构后 |
| --- | --- | --- |
| 非作者更新/删除/发布文章 | 1302 | 1302 |
| 非作者删除评论 | 1403 | 1403 |
| 未登录调用受保护接口 | 1002 | 1002 |
| 普通用户访问管理员接口 | 1003 | 1003 |
| 旧密码错误 | 1102 | 1102 |

应用层测试覆盖：[service_test.go](/Users/staff/code/go/blog-system/blog/internal/application/content/service_test.go)、[service_test.go](/Users/staff/code/go/blog-system/blog/internal/application/community/service_test.go)。

## 4. 统计与列表对比

| 行为 | 重构前 | 重构后 |
| --- | --- | --- |
| 公开文章列表只返回已发布 | 是 | 是 |
| 游标/分页 `last_id` 语义 | 是 | 是 |
| 开放列表排除删除 | 是 | 是 |
| 热榜公式 `view + like + comment` | 是 | 是 |
| 评论热度 `like + reply` | 是 | 是 |
| 点赞幂等（重复请求不重复写库/计数） | 是 | 是 |
| 游客不记录浏览历史但计数 | 是 | 是 |
| 自赞不产生通知 | 是 | 是 |

## 5. 验证结论

- `go test ./...` 通过，覆盖 Domain、Application、Repository、Interfaces 契约。
- `go vet ./...` 无静态检查问题。
- `make check-arch` 通过，Domain/Application 未依赖 Gin、GORM、Redis、MongoDB、Kafka、MinIO 或 gRPC 框架。
- `go build ./...` 通过，`server`、`grpc`、`kafka-consume` 三个启动方式保持可用。
