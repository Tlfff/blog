package model

import "errors"

// Status 是文章状态枚举类型。
type Status int8

// 文章状态取值与查询过滤值
const (
	All              = -2 // 全部（含删除）
	AllExceptDeleted = -1 // 全部（不含删除）
	Deleted          = 1  // 已删除
	Draft            = 2  // 草稿
	Published        = 3  // 已发表

)

// 实现fmt 包里的fmt.Stringer 接口，这样调用fmt的打印函数时会自动输出成对应的文字
func (r Status) String() string {
	// 1. 按状态值返回对应中文描述
	switch r {
	case Deleted:
		return "已删除"
	case All:
		return "全部"
	case Draft:
		return "草稿"
	case Published:
		return "已发表"
	// 2. 未登记的状态值统一返回未知状态
	default:
		return "未知状态"
	}
}

// 校验状态ID是否为合法的文章状态，非法时返回错误
func FindStatusById(statusId int) error {
	// 1. 转换为状态枚举类型
	r := Status(statusId)
	// 2. 仅允许已删除、草稿、已发表三种实体状态
	switch r {
	case Deleted, Draft, Published:
		return nil
	// 3. 其余取值视为非法状态
	default:
		return errors.New("不存在该状态")
	}
}
