package model

import "errors"

// Role 是用户角色枚举类型。
type Role int8

// 用户角色取值
const (
	RoleUser  = 1 // 普通用户
	RoleAdmin = 2 // 管理员
)

// 实现 fmt.Stringer 接口，使角色值在格式化打印时输出对应中文描述
func (r Role) String() string {
	// 1. 按角色值返回对应中文描述
	switch r {
	case RoleUser:
		return "用户"
	case RoleAdmin:
		return "管理员"
	// 2. 未登记的角色值统一返回未知角色
	default:
		return "未知角色"
	}
}

// 校验角色ID是否为合法的用户角色，非法时返回错误
func FindRoleById(roleId int) error {
	// 1. 转换为角色枚举类型
	r := Role(roleId)
	// 2. 仅允许普通用户与管理员两种角色
	switch r {
	case RoleUser, RoleAdmin:
		return nil
	// 3. 其余取值视为非法角色
	default:
		return errors.New("不存在该角色")
	}
}
