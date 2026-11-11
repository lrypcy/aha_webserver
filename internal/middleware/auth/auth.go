// Package auth 提供基于角色的授权。
// 当前实现为不依赖外部组件的轻量 RBAC；init/casbin.sql 保留为将来接入 Casbin 的参考。
package auth

import (
	"github.com/gofiber/fiber/v2"
	"github.com/lrypcy/aha_webserver/internal/middleware/jwt"
	"github.com/lrypcy/aha_webserver/internal/model"
	"github.com/lrypcy/aha_webserver/internal/response"
)

// Authorizer 是无状态的授权器。
type Authorizer struct{}

// New 构造授权器。
func New() *Authorizer { return &Authorizer{} }

// RequireRole 要求请求具备指定角色之一；admin 隐式放行。
func (a *Authorizer) RequireRole(roles ...string) fiber.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(c *fiber.Ctx) error {
		role, _ := c.Locals(jwt.RoleLocalKey).(string)
		if role == "" {
			return response.Fail(c, fiber.StatusUnauthorized, response.CodeUnauthorized, "authentication required")
		}
		if role == model.RoleAdmin {
			return c.Next()
		}
		if _, ok := allowed[role]; !ok {
			return response.Fail(c, fiber.StatusForbidden, response.CodeForbidden, "insufficient role")
		}
		return c.Next()
	}
}

// UserID 返回当前请求的用户 ID，未认证时返回 0。
func UserID(c *fiber.Ctx) uint {
	v, _ := c.Locals(jwt.UserIDLocalKey).(uint)
	return v
}

// Role 返回当前请求的角色。
func Role(c *fiber.Ctx) string {
	v, _ := c.Locals(jwt.RoleLocalKey).(string)
	return v
}

// IsAdmin 报告当前请求是否为管理员。
func IsAdmin(c *fiber.Ctx) bool { return Role(c) == model.RoleAdmin }
