// Package router 统一注册 /api/v1 下的业务路由。
package router

import (
	"github.com/gofiber/fiber/v2"
	authctrl "github.com/lrypcy/aha_webserver/internal/controller/auth"
	jobctrl "github.com/lrypcy/aha_webserver/internal/controller/job"
	taskctrl "github.com/lrypcy/aha_webserver/internal/controller/task"
	"github.com/lrypcy/aha_webserver/internal/middleware/auth"
	"github.com/lrypcy/aha_webserver/internal/middleware/jwt"
	"gorm.io/gorm"
)

// Deps 是路由层所需依赖，由 internal/app 注入。
type Deps struct {
	DB     *gorm.DB
	Tokens *jwt.Manager
	Authz  *auth.Authorizer
}

// InitRouting 注册所有业务路由。
func InitRouting(app *fiber.App, d Deps) {
	api := app.Group("/api/v1")
	guard := d.Tokens.Middleware()

	// 公开：注册 / 登录；受保护：其余全部接口。
	// 统一在此处注入 guard，避免像旧代码那样遗漏某个路由的鉴权。
	authctrl.NewHandler(d.DB, d.Tokens).Routes(api, guard)
	jobctrl.NewHandler(d.DB, d.Authz).Routes(api, guard)
	taskctrl.NewHandler(d.DB).Routes(api, guard)
}
