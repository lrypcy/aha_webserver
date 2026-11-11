package response

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// 业务错误码。0 表示成功，1xxxx 为通用错误，2xxxx 为认证/资源错误。
const (
	CodeOK                 = 0
	CodeBadRequest         = 10001
	CodeUnauthorized       = 10002
	CodeForbidden          = 10003
	CodeNotFound           = 10004
	CodeConflict           = 10005
	CodeTooManyRequests    = 10006
	CodeInternal           = 10007
	CodeInvalidCredentials = 20001
	CodeInvalidToken       = 20002
	CodeUserExists         = 20003
	CodeEmailExists        = 20004
)

// AppError 携带 HTTP 状态、业务错误码与可选的内部错误。
type AppError struct {
	Status  int
	Code    int
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap 让 errors.Is / errors.As 能穿透到底层错误。
func (e *AppError) Unwrap() error { return e.Err }

// New 构造一个业务错误。
func New(status, code int, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}

// Wrap 构造一个包装了内部原因的业务错误。
func Wrap(status, code int, message string, err error) *AppError {
	return &AppError{Status: status, Code: code, Message: message, Err: err}
}

func BadRequest(message string) *AppError {
	return New(fiber.StatusBadRequest, CodeBadRequest, message)
}
func Unauthorized(message string) *AppError {
	return New(fiber.StatusUnauthorized, CodeUnauthorized, message)
}
func Forbidden(message string) *AppError {
	return New(fiber.StatusForbidden, CodeForbidden, message)
}
func NotFound(message string) *AppError {
	return New(fiber.StatusNotFound, CodeNotFound, message)
}
func Conflict(message string) *AppError {
	return New(fiber.StatusConflict, CodeConflict, message)
}
func Internal(message string, err error) *AppError {
	return Wrap(fiber.StatusInternalServerError, CodeInternal, message, err)
}

// ErrorHandler 是 Fiber 的全局错误处理器：对外只暴露脱敏消息，细节写日志。
func ErrorHandler(c *fiber.Ctx, err error) error {
	var appErr *AppError
	if errors.As(err, &appErr) {
		if appErr.Status >= http.StatusInternalServerError && appErr.Err != nil {
			slog.Error("request failed", "error", appErr.Err, "path", c.Path(), "request_id", requestID(c))
		}
		return Fail(c, appErr.Status, appErr.Code, appErr.Message)
	}

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		switch fiberErr.Code {
		case fiber.StatusNotFound:
			return Fail(c, fiberErr.Code, CodeNotFound, "resource not found")
		case fiber.StatusBadRequest:
			return Fail(c, fiberErr.Code, CodeBadRequest, "bad request")
		case fiber.StatusTooManyRequests:
			return Fail(c, fiberErr.Code, CodeTooManyRequests, "too many requests")
		case fiber.StatusUnauthorized:
			return Fail(c, fiberErr.Code, CodeUnauthorized, "unauthorized")
		default:
			return Fail(c, fiberErr.Code, CodeInternal, http.StatusText(fiberErr.Code))
		}
	}

	slog.Error("unhandled error", "error", err, "path", c.Path(), "request_id", requestID(c))
	return Fail(c, fiber.StatusInternalServerError, CodeInternal, "internal server error")
}
