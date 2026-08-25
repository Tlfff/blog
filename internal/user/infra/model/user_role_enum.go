package model

import "errors"

// Role 是用户角色枚举类型。
type Role int8

// 用户角色取值
const (
	RoleUser  = 1 // 普通用户
	RoleAdmin = 2 // 管理员
)

// String 返回用户角色中文描述。
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

// FindRoleByID 校验角色 ID 是否合法。
func FindRoleByID(roleID int) error {
	// 1. 转换为角色枚举类型
	r := Role(roleID)
	// 2. 仅允许普通用户与管理员两种角色
	switch r {
	case RoleUser, RoleAdmin:
		return nil
	// 3. 其余取值视为非法角色
	default:
		return errors.New("不存在该角色")
	}
}
