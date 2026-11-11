// Package response 定义统一的 HTTP 响应体与错误模型。
package response

import "github.com/gofiber/fiber/v2"

// RequestIDHeader 是请求追踪头。
const RequestIDHeader = "X-Request-ID"

// RequestIDLocalKey 是 requestid 中间件写入 c.Locals 的键。
const RequestIDLocalKey = "requestid"

// Body 是所有接口统一的响应信封。
type Body struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data"`
	RequestID string `json:"request_id,omitempty"`
}

// PageData 是分页列表的 data 载荷。
type PageData struct {
	Items    any   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

func requestID(c *fiber.Ctx) string {
	if v, ok := c.Locals(RequestIDLocalKey).(string); ok && v != "" {
		return v
	}
	return c.Get(RequestIDHeader)
}

func write(c *fiber.Ctx, status, code int, message string, data any) error {
	return c.Status(status).JSON(Body{
		Code:      code,
		Message:   message,
		Data:      data,
		RequestID: requestID(c),
	})
}

// OK 返回 200。
func OK(c *fiber.Ctx, data any) error {
	return write(c, fiber.StatusOK, CodeOK, "ok", data)
}

// Created 返回 201。
func Created(c *fiber.Ctx, data any) error {
	return write(c, fiber.StatusCreated, CodeOK, "created", data)
}

// Fail 返回统一的错误响应。
func Fail(c *fiber.Ctx, status, code int, message string) error {
	return write(c, status, code, message, nil)
}
