// Package dto 定义请求/响应数据传输对象。
// 约束： controller 只能把 DTO 绑定到请求，绝不直接绑定 model，避免 mass assignment。
package dto

import (
	"errors"
	"fmt"
	"net/mail"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/lrypcy/aha_webserver/internal/model"
)

// RegisterRequest 注册请求。
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate 手写校验（Fiber 的 BodyParser 不识别 binding/validate 标签）。
func (r RegisterRequest) Validate() error {
	u := strings.TrimSpace(r.Username)
	if len(u) < 3 || len(u) > 50 {
		return errors.New("username must be 3-50 characters")
	}
	if _, err := mail.ParseAddress(strings.TrimSpace(r.Email)); err != nil {
		return errors.New("email is invalid")
	}
	return validatePassword(r.Password)
}

func validatePassword(p string) error {
	if len([]byte(p)) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	if len([]byte(p)) > 72 {
		return errors.New("password must be at most 72 bytes (bcrypt limit)")
	}
	return nil
}

// LoginRequest 登录请求。
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (r LoginRequest) Validate() error {
	if strings.TrimSpace(r.Username) == "" || r.Password == "" {
		return errors.New("username and password are required")
	}
	return nil
}

// CreateJobRequest 创建 Job 请求。
type CreateJobRequest struct {
	Title   string         `json:"title"`
	Creator string         `json:"creator"`
	Config  map[string]any `json:"config"`
	Labels  []string       `json:"labels"`
}

func (r CreateJobRequest) Validate() error {
	if strings.TrimSpace(r.Title) == "" {
		return errors.New("title is required")
	}
	if len(r.Title) > 255 {
		return errors.New("title must be at most 255 characters")
	}
	return nil
}

// CreateTaskRequest 创建 Task 请求。
type CreateTaskRequest struct {
	JobID   uint           `json:"job_id"`
	Title   string         `json:"title"`
	Creator string         `json:"creator"`
	Config  map[string]any `json:"config"`
	Labels  []string       `json:"labels"`
}

func (r CreateTaskRequest) Validate() error {
	if r.JobID == 0 {
		return errors.New("job_id is required")
	}
	if strings.TrimSpace(r.Title) == "" {
		return errors.New("title is required")
	}
	return nil
}

// UpdateTaskStatusRequest 更新任务状态请求。
type UpdateTaskStatusRequest struct {
	Status    model.Status    `json:"status"`
	ErrorCode model.ErrorCode `json:"error_code"`
}

func (r UpdateTaskStatusRequest) Validate() error {
	if !r.Status.Valid() {
		return fmt.Errorf("status must be one of %v", model.AllStatuses())
	}
	return nil
}

// PageQuery 通用分页与过滤参数。
type PageQuery struct {
	Page     int
	PageSize int
	JobID    uint
	Status   *model.Status
}

// ParsePageQuery 从 query string 解析并做上下界收敛。
func ParsePageQuery(c *fiber.Ctx) PageQuery {
	q := PageQuery{
		Page:     atoiDefault(c.Query("page"), 1),
		PageSize: atoiDefault(c.Query("page_size"), 20),
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
	if v := c.Query("job_id"); v != "" {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			q.JobID = uint(n)
		}
	}
	if v := c.Query("status"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			s := model.Status(n)
			q.Status = &s
		}
	}
	return q
}

// Offset 返回 SQL OFFSET。
func (q PageQuery) Offset() int { return (q.Page - 1) * q.PageSize }

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
