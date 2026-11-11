// Package auth 提供注册、登录与当前用户信息接口。
package auth

import (
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/lrypcy/aha_webserver/internal/dto"
	authmw "github.com/lrypcy/aha_webserver/internal/middleware/auth"
	"github.com/lrypcy/aha_webserver/internal/middleware/jwt"
	"github.com/lrypcy/aha_webserver/internal/model"
	"github.com/lrypcy/aha_webserver/internal/response"
	"gorm.io/gorm"
)

// Handler 持有依赖，避免全局变量，便于测试注入内存数据库。
type Handler struct {
	db     *gorm.DB
	tokens *jwt.Manager
}

func NewHandler(db *gorm.DB, tokens *jwt.Manager) *Handler {
	return &Handler{db: db, tokens: tokens}
}

// Routes 注册认证相关路由。guard 为 JWT 中间件。
func (h *Handler) Routes(r fiber.Router, guard fiber.Handler) {
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Get("/me", guard, h.Me)
}

// Register 用户注册
// @Summary 注册新用户
// @Tags auth
// @Accept json
// @Produce json
// @Param body body dto.RegisterRequest true "注册信息"
// @Success 201 {object} response.Body
// @Failure 400 {object} response.Body
// @Failure 409 {object} response.Body
// @Router /register [post]
func (h *Handler) Register(c *fiber.Ctx) error {
	var req dto.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest("invalid request body")
	}
	if err := req.Validate(); err != nil {
		return response.BadRequest(err.Error())
	}

	username := strings.TrimSpace(req.Username)
	email := strings.TrimSpace(req.Email)

	var count int64
	if err := h.db.Model(&model.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return response.Internal("database error", err)
	}
	if count > 0 {
		return response.Fail(c, fiber.StatusConflict, response.CodeUserExists, "username already taken")
	}
	if err := h.db.Model(&model.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return response.Internal("database error", err)
	}
	if count > 0 {
		return response.Fail(c, fiber.StatusConflict, response.CodeEmailExists, "email already registered")
	}

	user := model.User{
		Username: username,
		Email:    email,
		Password: req.Password, // BeforeCreate 钩子负责 bcrypt 哈希
		Role:     model.RoleUser,
		IsActive: true,
	}
	if err := h.db.Create(&user).Error; err != nil {
		return response.Internal("create user failed", err)
	}
	return response.Created(c, user.Info())
}

// Login 用户登录并签发 JWT
// @Summary 登录并获取令牌
// @Tags auth
// @Accept json
// @Produce json
// @Param body body dto.LoginRequest true "登录信息"
// @Success 200 {object} response.Body
// @Failure 400 {object} response.Body
// @Failure 401 {object} response.Body
// @Router /login [post]
func (h *Handler) Login(c *fiber.Ctx) error {
	var req dto.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest("invalid request body")
	}
	if err := req.Validate(); err != nil {
		return response.BadRequest(err.Error())
	}

	var user model.User
	if err := h.db.Where("username = ?", strings.TrimSpace(req.Username)).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.Fail(c, fiber.StatusUnauthorized, response.CodeInvalidCredentials, "invalid credentials")
		}
		return response.Internal("database error", err)
	}
	if !user.IsActive {
		return response.Fail(c, fiber.StatusForbidden, response.CodeForbidden, "account is disabled")
	}
	if !user.VerifyPassword(req.Password) {
		return response.Fail(c, fiber.StatusUnauthorized, response.CodeInvalidCredentials, "invalid credentials")
	}

	now := time.Now()
	if err := h.db.Model(&user).Update("last_login", now).Error; err != nil {
		// 最近登录时间属于可降级信息，不影响登录主流程。
		slog.Warn("update last_login failed", "error", err, "user_id", user.ID)
	}
	user.MarkLogin(now)

	token, err := h.tokens.Generate(user.ID, user.Username, user.Role)
	if err != nil {
		return response.Internal("could not issue token", err)
	}
	return response.OK(c, fiber.Map{
		"token":      token,
		"token_type": "Bearer",
		"expires_in": int(h.tokens.Expiration().Seconds()),
		"user":       user.Info(),
	})
}

// Me 返回当前登录用户
// @Summary 获取当前用户
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Body
// @Failure 401 {object} response.Body
// @Router /me [get]
func (h *Handler) Me(c *fiber.Ctx) error {
	var user model.User
	if err := h.db.First(&user, authmw.UserID(c)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound("user not found")
		}
		return response.Internal("database error", err)
	}
	return response.OK(c, user.Info())
}
