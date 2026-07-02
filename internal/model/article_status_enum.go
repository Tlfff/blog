package model

import "errors"

type Status int8

const (
	All       = 0  //全部，除已删除的
	Draft     = 1  // 草稿
	Published = 2  // 已发表
	Deleted   = 99 // 已删除
)

// 实现fmt 包里的fmt.Stringer 接口，这样调用fmt的打印函数时会自动输出成对应的文字
func (r Status) String() string {
	switch r {
	case Deleted:
		return "已删除"
	case All:
		return "全部"
	case Draft:
		return "草稿"
	case Published:
		return "已发表"
	default:
		return "未知状态"
	}
}

func FindStatusById(statusId int) error {
	r := Status(statusId)
	switch r {
	case Deleted, Draft, Published:
		return nil
	default:
		return errors.New("不存在该状态")
	}
}
