// Package task 提供 Task 的创建、查询与状态流转接口。
package task

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/lrypcy/aha_webserver/internal/dto"
	"github.com/lrypcy/aha_webserver/internal/middleware/jwt"
	"github.com/lrypcy/aha_webserver/internal/model"
	"github.com/lrypcy/aha_webserver/internal/response"
	"gorm.io/gorm"
)

// Handler 持有数据库连接。
type Handler struct{ db *gorm.DB }

func NewHandler(db *gorm.DB) *Handler { return &Handler{db: db} }

// Routes 注册 /tasks 系列路由。
func (h *Handler) Routes(r fiber.Router, guard fiber.Handler) {
	r.Post("/tasks", guard, h.Create)
	r.Get("/tasks", guard, h.List)
	r.Get("/tasks/:id", guard, h.Get)
	r.Patch("/tasks/:id/status", guard, h.UpdateStatus)
}

// Create 创建 Task
// @Summary 创建 Task
// @Tags tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body dto.CreateTaskRequest true "Task 信息"
// @Success 201 {object} response.Body
// @Failure 400 {object} response.Body
// @Router /tasks [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	var req dto.CreateTaskRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest("invalid request body")
	}
	if err := req.Validate(); err != nil {
		return response.BadRequest(err.Error())
	}

	var parent model.Job
	if err := h.db.Select("id").First(&parent, req.JobID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.BadRequest("job_id does not exist")
		}
		return response.Internal("database error", err)
	}

	task := model.Task{
		JobID:   req.JobID,
		Title:   strings.TrimSpace(req.Title),
		Creator: creatorOf(c, req.Creator),
		Status:  model.StatusPending,
		Config:  req.Config,
		Labels:  req.Labels,
	}
	if err := h.db.Create(&task).Error; err != nil {
		return response.Internal("create task failed", err)
	}
	return response.Created(c, task)
}

// List 分页查询 Task
// @Summary 分页列出 Task
// @Tags tasks
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页条数" default(20)
// @Param job_id query int false "按 Job 过滤"
// @Param status query int false "按状态过滤"
// @Success 200 {object} response.Body
// @Router /tasks [get]
func (h *Handler) List(c *fiber.Ctx) error {
	q := dto.ParsePageQuery(c)
	scoped := func() *gorm.DB {
		tx := h.db.Model(&model.Task{})
		if q.JobID != 0 {
			tx = tx.Where("job_id = ?", q.JobID)
		}
		if q.Status != nil {
			tx = tx.Where("status = ?", *q.Status)
		}
		return tx
	}

	var total int64
	if err := scoped().Count(&total).Error; err != nil {
		return response.Internal("count tasks failed", err)
	}
	var tasks []model.Task
	if err := scoped().Offset(q.Offset()).Limit(q.PageSize).Order("id DESC").Find(&tasks).Error; err != nil {
		return response.Internal("list tasks failed", err)
	}
	return response.OK(c, response.PageData{
		Items: tasks, Total: total, Page: q.Page, PageSize: q.PageSize,
	})
}

// Get 查询 Task 详情
// @Summary 获取 Task 详情
// @Tags tasks
// @Produce json
// @Security BearerAuth
// @Param id path int true "Task ID"
// @Success 200 {object} response.Body
// @Failure 404 {object} response.Body
// @Router /tasks/{id} [get]
func (h *Handler) Get(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return response.BadRequest("id must be a positive integer")
	}
	var task model.Task
	if err := h.db.First(&task, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound("task not found")
		}
		return response.Internal("database error", err)
	}
	return response.OK(c, task)
}

// UpdateStatus 更新 Task 状态
// @Summary 更新 Task 状态
// @Tags tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Task ID"
// @Param body body dto.UpdateTaskStatusRequest true "目标状态"
// @Success 200 {object} response.Body
// @Failure 400 {object} response.Body
// @Failure 404 {object} response.Body
// @Router /tasks/{id}/status [patch]
func (h *Handler) UpdateStatus(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return response.BadRequest("id must be a positive integer")
	}
	var req dto.UpdateTaskStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest("invalid request body")
	}
	if err := req.Validate(); err != nil {
		return response.BadRequest(err.Error())
	}

	var task model.Task
	if err := h.db.First(&task, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound("task not found")
		}
		return response.Internal("database error", err)
	}
	if err := h.db.Model(&task).Updates(map[string]any{
		"status":     req.Status,
		"error_code": req.ErrorCode,
	}).Error; err != nil {
		return response.Internal("update task failed", err)
	}
	task.Status = req.Status
	task.ErrorCode = req.ErrorCode
	return response.OK(c, task)
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
