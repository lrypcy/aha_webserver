// Package job 提供 Job 的增删改查接口。
package job

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/lrypcy/aha_webserver/internal/dto"
	authmw "github.com/lrypcy/aha_webserver/internal/middleware/auth"
	"github.com/lrypcy/aha_webserver/internal/middleware/jwt"
	"github.com/lrypcy/aha_webserver/internal/model"
	"github.com/lrypcy/aha_webserver/internal/response"
	"gorm.io/gorm"
)

// Handler 持有数据库连接与授权器。
type Handler struct {
	db    *gorm.DB
	authz *authmw.Authorizer
}

func NewHandler(db *gorm.DB, authz *authmw.Authorizer) *Handler {
	return &Handler{db: db, authz: authz}
}

// Routes 注册 /jobs 系列路由。
func (h *Handler) Routes(r fiber.Router, guard fiber.Handler) {
	r.Post("/jobs", guard, h.Create)
	r.Get("/jobs", guard, h.List)
	r.Get("/jobs/:id", guard, h.Get)
	r.Delete("/jobs/:id", guard, h.authz.RequireRole(model.RoleAdmin), h.Delete)
}

// Create 创建 Job
// @Summary 创建 Job
// @Tags jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body dto.CreateJobRequest true "Job 信息"
// @Success 201 {object} response.Body
// @Failure 400 {object} response.Body
// @Router /jobs [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	var req dto.CreateJobRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest("invalid request body")
	}
	if err := req.Validate(); err != nil {
		return response.BadRequest(err.Error())
	}

	job := model.Job{
		Title:   strings.TrimSpace(req.Title),
		Creator: creatorOf(c, req.Creator),
		Status:  model.StatusPending,
		Config:  req.Config,
		Labels:  req.Labels,
	}
	if err := h.db.Create(&job).Error; err != nil {
		return response.Internal("create job failed", err)
	}
	return response.Created(c, job)
}

// List 分页查询 Job
// @Summary 分页列出 Job
// @Tags jobs
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页条数" default(20)
// @Param status query int false "按状态过滤"
// @Success 200 {object} response.Body
// @Router /jobs [get]
func (h *Handler) List(c *fiber.Ctx) error {
	q := dto.ParsePageQuery(c)
	scoped := func() *gorm.DB {
		tx := h.db.Model(&model.Job{})
		if q.Status != nil {
			tx = tx.Where("status = ?", *q.Status)
		}
		return tx
	}

	var total int64
	if err := scoped().Count(&total).Error; err != nil {
		return response.Internal("count jobs failed", err)
	}
	var jobs []model.Job
	if err := scoped().Offset(q.Offset()).Limit(q.PageSize).Order("id DESC").Find(&jobs).Error; err != nil {
		return response.Internal("list jobs failed", err)
	}
	return response.OK(c, response.PageData{
		Items: jobs, Total: total, Page: q.Page, PageSize: q.PageSize,
	})
}

// Get 查询 Job 详情（含 Tasks 与 Result）
// @Summary 获取 Job 详情
// @Tags jobs
// @Produce json
// @Security BearerAuth
// @Param id path int true "Job ID"
// @Success 200 {object} response.Body
// @Failure 404 {object} response.Body
// @Router /jobs/{id} [get]
func (h *Handler) Get(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return response.BadRequest("id must be a positive integer")
	}
	var job model.Job
	if err := h.db.Preload("Tasks").Preload("Result").First(&job, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound("job not found")
		}
		return response.Internal("database error", err)
	}
	return response.OK(c, job)
}

// Delete 软删除 Job 及其 Tasks（仅管理员）
// @Summary 删除 Job
// @Tags jobs
// @Produce json
// @Security BearerAuth
// @Param id path int true "Job ID"
// @Success 200 {object} response.Body
// @Failure 403 {object} response.Body
// @Failure 404 {object} response.Body
// @Router /jobs/{id} [delete]
func (h *Handler) Delete(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return response.BadRequest("id must be a positive integer")
	}
	var existing model.Job
	if err := h.db.Select("id").First(&existing, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound("job not found")
		}
		return response.Internal("database error", err)
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("job_id = ?", id).Delete(&model.Task{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Job{}, id).Error
	}); err != nil {
		return response.Internal("delete job failed", err)
	}
	return response.OK(c, fiber.Map{"id": id, "deleted": true})
}

func creatorOf(c *fiber.Ctx, provided string) string {
	if s := strings.TrimSpace(provided); s != "" {
		return s
	}
	if name, ok := c.Locals(jwt.UsernameLocalKey).(string); ok && name != "" {
		return name
	}
	return "unknown"
}
