// Package app 负责装配 Fiber 应用：全局中间件、探针、文档与业务路由。
package app

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/lrypcy/aha_webserver/internal/config"
	"github.com/lrypcy/aha_webserver/internal/controller/health"
	"github.com/lrypcy/aha_webserver/internal/middleware/auth"
	"github.com/lrypcy/aha_webserver/internal/middleware/jwt"
	"github.com/lrypcy/aha_webserver/internal/response"
	"github.com/lrypcy/aha_webserver/internal/router"
	fiberSwagger "github.com/swaggo/fiber-swagger"
	"gorm.io/gorm"
)

// Deps 是应用装配所需的全部依赖。
type Deps struct {
	Config    *config.Config
	DB        *gorm.DB
	Tokens    *jwt.Manager
	Version   string
	StartedAt time.Time
}

// New 构造并返回完整应用。
func New(d Deps) *fiber.App {
	cfg := d.Config
	application := fiber.New(fiber.Config{
		AppName:               "aha_webserver " + d.Version,
		ErrorHandler:          response.ErrorHandler,
		ReadTimeout:           cfg.Server.ReadTimeout,
		WriteTimeout:          cfg.Server.WriteTimeout,
		BodyLimit:             cfg.Server.BodyLimit,
		DisableStartupMessage: cfg.IsProduction(),
	})

	// 顺序：requestid → 日志 → panic 恢复 → 安全头 → CORS → 限流。
	application.Use(requestid.New(requestid.Config{Header: response.RequestIDHeader}))
	application.Use(logger.New(logger.Config{
		Format: "${time} | ${status} | ${latency} | ${ip} | ${method} ${path} | req=${locals:requestid}\n",
	}))
	application.Use(recover.New())
	// Swagger UI 依赖内联脚本，关闭 CSP；其余安全头仍然生效。
	application.Use(helmet.New(helmet.Config{ContentSecurityPolicy: ""}))
	application.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORS.Origins,
		AllowMethods:     "GET,POST,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Request-ID",
		ExposeHeaders:    "X-Request-ID",
		AllowCredentials: false,
		MaxAge:           300,
	}))
	application.Use(limiter.New(limiter.Config{
		Max:        cfg.RateLimit.Max,
		Expiration: cfg.RateLimit.Expiration,
		Next: func(c *fiber.Ctx) bool {
			p := c.Path()
			return p == "/health" || p == "/ready" || strings.HasPrefix(p, "/swagger")
		},
		LimitReached: func(c *fiber.Ctx) error {
			return response.Fail(c, fiber.StatusTooManyRequests, response.CodeTooManyRequests, "too many requests")
		},
	}))

	health.NewHandler(d.DB, d.Version, d.StartedAt).Routes(application)
	application.Get("/swagger/*", fiberSwagger.WrapHandler)

	router.InitRouting(application, router.Deps{
		DB:     d.DB,
		Tokens: d.Tokens,
		Authz:  auth.New(),
	})
	return application
}
