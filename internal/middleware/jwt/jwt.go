// Package jwt 提供 HS256 令牌的签发、解析与 Fiber 认证中间件。
// 通过 Manager 注入密钥，避免包级可变全局状态。
package jwt

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/lrypcy/aha_webserver/internal/response"
)

const (
	// UserIDLocalKey / RoleLocalKey / UsernameLocalKey 是写入 c.Locals 的键。
	UserIDLocalKey   = "userID"
	RoleLocalKey     = "role"
	UsernameLocalKey = "username"
)

// Claims 是令牌载荷。
type Claims struct {
	UserID   uint   `json:"user_id"`
	Role     string `json:"role"`
	Username string `json:"username"`
	gojwt.RegisteredClaims
}

// Manager 持有签名密钥与签发策略。
type Manager struct {
	secret     []byte
	expiration time.Duration
	issuer     string
}

// New 构造 Manager。
func New(secret string, expiration time.Duration, issuer string) (*Manager, error) {
	if len(secret) < 16 {
		return nil, errors.New("jwt secret must be at least 16 characters")
	}
	if expiration <= 0 {
		return nil, errors.New("jwt expiration must be positive")
	}
	if issuer == "" {
		issuer = "aha_webserver"
	}
	return &Manager{secret: []byte(secret), expiration: expiration, issuer: issuer}, nil
}

// Expiration 返回令牌有效期。
func (m *Manager) Expiration() time.Duration { return m.expiration }

// Generate 签发令牌。
func (m *Manager) Generate(userID uint, username, role string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		Role:     role,
		Username: username,
		RegisteredClaims: gojwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   strconv.FormatUint(uint64(userID), 10),
			ExpiresAt: gojwt.NewNumericDate(now.Add(m.expiration)),
			IssuedAt:  gojwt.NewNumericDate(now),
			NotBefore: gojwt.NewNumericDate(now.Add(-time.Minute)),
		},
	}
	token, err := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return token, nil
}

// Parse 解析并校验令牌，同时固定算法白名单防止算法混淆。
func (m *Manager) Parse(raw string) (*Claims, error) {
	claims := &Claims{}
	token, err := gojwt.ParseWithClaims(raw, claims,
		func(t *gojwt.Token) (any, error) {
			if _, ok := t.Method.(*gojwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return m.secret, nil
		},
		gojwt.WithValidMethods([]string{gojwt.SigningMethodHS256.Alg()}),
		gojwt.WithIssuer(m.issuer),
		gojwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// ExtractBearer 从 Authorization 头提取 Bearer 令牌。
func ExtractBearer(header string) (string, error) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", errors.New("Authorization header must be 'Bearer <token>'")
	}
	return parts[1], nil
}

// Middleware 校验请求令牌并把身份写入 Locals。
func (m *Manager) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		raw, err := ExtractBearer(c.Get("Authorization"))
		if err != nil {
			return response.Fail(c, fiber.StatusUnauthorized, response.CodeInvalidToken, err.Error())
		}
		claims, err := m.Parse(raw)
		if err != nil {
			return response.Fail(c, fiber.StatusUnauthorized, response.CodeInvalidToken, "invalid or expired token")
		}
		c.Locals(UserIDLocalKey, claims.UserID)
		c.Locals(RoleLocalKey, claims.Role)
		c.Locals(UsernameLocalKey, claims.Username)
		return c.Next()
	}
}
