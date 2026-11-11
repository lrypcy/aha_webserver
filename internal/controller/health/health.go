// Package health 提供存活与就绪探针。
package health

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/lrypcy/aha_webserver/internal/response"
	"gorm.io/gorm"
)

// Handler 持有数据库句柄与启动元信息。
type Handler struct {
	db        *gorm.DB
	version   string
	startedAt time.Time
}

func NewHandler(db *gorm.DB, version string, startedAt time.Time) *Handler {
	return &Handler{db: db, version: version, startedAt: startedAt}
}

// Routes 注册探针（不受限流与鉴权影响）。
func (h *Handler) Routes(r fiber.Router) {
	r.Get("/health", h.Liveness)
	r.Get("/ready", h.Readiness)
}

// Liveness 存活探针。
// 注意：探针挂在根路径而非 /api/v1 下，因此不写 swag 的 @Router 注解，避免文档路径错误。
func (h *Handler) Liveness(c *fiber.Ctx) error {
	return response.OK(c, fiber.Map{
		"status":  "ok",
		"version": h.version,
		"uptime":  time.Since(h.startedAt).String(),
	})
}

// Readiness 就绪探针（含数据库连通性）。
// 同 Liveness，探针路径不受 @BasePath 约束，故不生成 swagger 定义。
func (h *Handler) Readiness(c *fiber.Ctx) error {
	if h.db == nil {
		return response.Fail(c, fiber.StatusServiceUnavailable, response.CodeInternal, "database not initialized")
	}
	sqlDB, err := h.db.DB()
	if err != nil {
		return response.Fail(c, fiber.StatusServiceUnavailable, response.CodeInternal, "database unavailable")
	}
	if err := sqlDB.Ping(); err != nil {
		return response.Fail(c, fiber.StatusServiceUnavailable, response.CodeInternal, "database unavailable")
	}
	return response.OK(c, fiber.Map{"status": "ready", "database": "up"})
}
