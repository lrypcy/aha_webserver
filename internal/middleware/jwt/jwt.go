package jwt

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lrypcy/aha_webserver/internal/database"
)

type Claims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

var secretKey string
var expiration time.Duration

func Init(secret string, exp time.Duration) error {
	type Config struct {
		Value string
	}
	var cfg Config
	
	// 从数据库查询JWT密钥配置
	result := database.DB().Table("sys_config").
		Select("value").
		Where("key = ?", "jwt_secret").
		First(&cfg)
	
	if result.Error != nil {
		return result.Error
	}
	
	secretKey = cfg.Value

	// 从数据库查询JWT过期时间配置
	result = database.DB().Table("sys_config").
		Select("value").
		Where("key = ?", "jwt_expiration").
		First(&cfg)
	
	if result.Error != nil {
		return result.Error
	}

	// 解析持续时间
	var err error
	expiration, err = time.ParseDuration(cfg.Value)
	if err != nil {
		return err
	}
	
	return nil
}

func GenerateToken(userID uint) (string, error) {
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretKey))
}

func Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		tokenString := c.Get("Authorization")
		if tokenString == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authorization header is missing",
			})
		}

		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(secretKey), nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token",
			})
		}

		claims, ok := token.Claims.(*Claims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token claims",
			})
		}

		c.Locals("userID", claims.UserID)
		return c.Next()
	}
}
