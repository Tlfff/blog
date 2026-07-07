package common

import (
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// 昵称不能纯数字，空字符串也拦截
func notOnlyNumber(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	// 空字符串直接校验失败
	if s == "" {
		return false
	}
	// 遍历全部字符，全数字则失败
	for _, r := range s {
		if r < '0' || r > '9' {
			return true
		}
	}
	return false
}

// 全局注册所有自定义校验规则
func InitValidator() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = v.RegisterValidation("not_only_number", notOnlyNumber)
	}
}
