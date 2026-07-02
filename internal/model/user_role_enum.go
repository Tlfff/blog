package model

import "errors"

type Role int8

const (
	RoleUser  = 1
	RoleAdmin = 2
)

func (r Role) String() string {
	switch r {
	case RoleUser:
		return "用户"
	case RoleAdmin:
		return "管理员"
	default:
		return "未知角色"
	}
}

func FindRoleById(roleId int) error {
	r := Role(roleId)
	switch r {
	case RoleUser, RoleAdmin:
		return nil
	default:
		return errors.New("不存在该角色")
	}
}
