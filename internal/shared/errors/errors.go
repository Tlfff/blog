// Package errors 提供统一的领域错误与应用错误类型。
package errors

// DomainError 表示领域层业务规则失败。
type DomainError struct {
	Code    int
	Message string
	Cause   error
}

func NewDomainError(code int, message string) *DomainError {
	return &DomainError{Code: code, Message: message}
}

func (e *DomainError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *DomainError) Unwrap() error {
	return e.Cause
}

// ApplicationError 表示应用层编排或权限校验失败。
type ApplicationError struct {
	Code    int
	Message string
	Cause   error
}

func NewApplicationError(code int, message string) *ApplicationError {
	return &ApplicationError{Code: code, Message: message}
}

func (e *ApplicationError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *ApplicationError) Unwrap() error {
	return e.Cause
}
