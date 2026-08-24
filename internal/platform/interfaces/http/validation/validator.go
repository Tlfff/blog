package validation

import (
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// 昵称不能纯数字，空字符串也拦截
// 校验字符串不能为纯数字，空字符串同样拦截
// 作为 validator 的 not_only_number 规则实现
func notOnlyNumber(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	// 1. 空字符串直接校验失败
	// 空字符串直接校验失败
	if s == "" {
		return false
	}
	// 2. 出现任意非数字字符即通过校验
	// 遍历全部字符，全数字则失败
	for _, r := range s {
		if r < '0' || r > '9' {
			return true
		}
	}
	// 3. 全为数字，校验失败
	return false
}

// InitValidator 全局注册所有自定义校验规则。
// 向 Gin 的校验引擎注册全部自定义校验规则
// 需在路由注册前调用一次
func InitValidator() {
	// 1. 取出 Gin 底层的 validator 引擎
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		// 2. 注册 not_only_number 规则
		_ = v.RegisterValidation("not_only_number", notOnlyNumber)
	}
}
