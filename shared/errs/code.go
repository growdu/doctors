// Package errs 定义业务错误码与统一错误类型。
//
// 设计要点：
//   - 错误码分两类：业务码（5 位）与系统码（6 位）。
//   - Code 关联 HTTP 状态码，便于中间件统一映射。
//   - Error 实现 error 接口，支持 errors.Is/As 链式追溯。
package errs

import (
	"errors"
	"fmt"
	"net/http"
)

// Code 是业务/系统错误码。0 表示成功。
type Code int

const (
	// CodeOK 成功。
	CodeOK Code = 0
	// CodeParamInvalid 参数无效。
	CodeParamInvalid Code = 10001
	// CodeUnauthorized 未登录或 token 无效。
	CodeUnauthorized Code = 11001
	// CodeForbidden 已登录但权限不足。
	CodeForbidden Code = 11002
	// CodeAdminForbidden 角色不在 admin API 白名单（如 viewer 调 force-cancel）。
	// admin plan 新增（2026-09-24）。
	CodeAdminForbidden Code = 11003
	// CodeNotFound 资源不存在。
	CodeNotFound Code = 12001
	// CodeConflict 资源冲突（如重复创建）。
	CodeConflict Code = 12002
	// CodeRateLimit 触发限流。
	CodeRateLimit Code = 13001
	// CodeUnprocessable 业务校验失败（如 escort 不在候选、不在可用时段等可恢复的业务错误）。
	CodeUnprocessable Code = 14001
	// CodeGone 资源已过期（如 escort_pending_expire_at 已过）。
	CodeGone Code = 14002
	// CodeUpstreamUnavailable 上游服务不可用（admin 调 order / refund / escort 失败）。
	// admin plan 新增（2026-09-24）。
	CodeUpstreamUnavailable Code = 15003

	// CodeInternal 服务器内部错误。
	CodeInternal Code = 500000
	// CodeUnavailable 服务暂时不可用。
	CodeUnavailable Code = 500001
)

// httpStatus 映射 Code 到 HTTP 状态码。
func (c Code) HTTPStatus() int {
	switch c {
	case CodeOK:
		return http.StatusOK
	case CodeParamInvalid:
		return http.StatusBadRequest
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeAdminForbidden:
		return http.StatusForbidden
	case CodeNotFound:
		return http.StatusNotFound
	case CodeConflict:
		return http.StatusConflict
	case CodeRateLimit:
		return http.StatusTooManyRequests
	case CodeUnprocessable:
		return http.StatusUnprocessableEntity
	case CodeGone:
		return http.StatusGone
	case CodeUpstreamUnavailable:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

// IsSuccess 判定是否为成功码（0）。
func (c Code) IsSuccess() bool { return c == CodeOK }

// String 返回错误码的数字字符串。
func (c Code) String() string { return fmt.Sprintf("%d", int(c)) }

// Error 是统一错误类型。
type Error struct {
	Code   Code
	Msg    string
	Cause  error
}

// Error 实现 error 接口。
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Msg, e.Cause)
	}
	return e.Msg
}

// Unwrap 让 errors.Is / errors.As 可穿透到 Cause。
func (e *Error) Unwrap() error { return e.Cause }

// New 构造一个新错误。
func New(code Code, msg string) *Error {
	return &Error{Code: code, Msg: msg}
}

// Wrap 包装一个底层错误。
func Wrap(code Code, msg string, cause error) *Error {
	return &Error{Code: code, Msg: msg, Cause: cause}
}

// As 强类型转换为 *Error。
func As(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}