## Purpose

为博客系统提供统一的用户身份、认证会话、个人资料和调用方身份能力；阶段一由单体内 Identity 领域模块提供，阶段二由 Identity Service 承载，使其他业务能力无需直接访问用户数据即可完成身份识别与用户信息查询。

## ADDED Requirements

### Requirement: User authentication and session management

身份服务 SHALL 支持用户注册、登录、退出登录、密码修改和会话失效，并 SHALL 保留当前系统对密码安全、记住我、多端会话和主动退出的可观察行为。

#### Scenario: Successful login creates a session
- **WHEN** 用户使用有效的手机号或昵称及密码登录
- **THEN** 系统返回可用于后续请求的会话凭证，并记录登录时间、登录 IP 和设备信息

#### Scenario: Invalid credentials are rejected
- **WHEN** 用户提供不存在的账号或错误密码
- **THEN** 系统拒绝登录并返回统一错误响应，且不得创建有效会话

#### Scenario: Logout invalidates the current session
- **WHEN** 已登录用户主动退出
- **THEN** 当前会话凭证立即失效，后续携带该凭证的请求不得继续访问受保护资源

### Requirement: Profile and account management

身份服务 SHALL 支持公开资料查询、当前用户资料查询、昵称/手机号/头像更新以及分步修改密码，并 SHALL 对敏感字段和用户归属进行校验。

#### Scenario: Public profile hides sensitive data
- **WHEN** 游客或其他用户查询公开主页
- **THEN** 系统只返回允许公开的用户 ID、昵称和头像等信息，不返回密码、手机号或会话数据

#### Scenario: User updates own profile
- **WHEN** 已登录用户提交合法的昵称、手机号或已确认的头像对象
- **THEN** 系统只更新该用户自己的资料，并返回成功结果

#### Scenario: Password change invalidates other sessions
- **WHEN** 用户通过有效的一次性改密凭证设置新密码
- **THEN** 新密码生效，当前会话可按既定策略保留，其他会话被失效

### Requirement: Internal user information access

身份服务 SHALL 提供受保护的用户基本信息和公开信息查询能力，供其他服务和开放 API 使用；不存在或已禁用的用户 SHALL 返回明确的业务错误。

#### Scenario: Existing user information is returned
- **WHEN** 合法的内部调用请求查询正常用户
- **THEN** 身份服务返回约定字段，并包含调用方所需的用户角色或公开资料信息

#### Scenario: Missing user is rejected
- **WHEN** 内部调用请求查询不存在或不可用的用户
- **THEN** 身份服务返回可识别的用户不存在错误，不返回空的伪造用户信息
