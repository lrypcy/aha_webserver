package auth

import (
	"github.com/lrypcy/aha_webserver/internal/middleware/jwt"

	"github.com/gofiber/fiber/v2"
	"github.com/lrypcy/aha_webserver/internal/database"
	"github.com/lrypcy/aha_webserver/internal/model"
	"golang.org/x/crypto/bcrypt"
)

func AuthenticateUser(username, password string) (*model.User, error) {
	var user model.User
	if err := database.DB().Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, err
	}
	
	return &user, nil
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request",
		})
	}

	// 这里添加你的用户验证逻辑
	user, err := AuthenticateUser(req.Username, req.Password)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid credentials",
		})
	}

	token, err := jwt.GenerateToken(user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not generate token",
		})
	}

	return c.JSON(fiber.Map{
		"token": token,
		"user":  user,
	})
} 

// 注册请求结构体
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=50"`
}

// RegisterUser 用户注册控制器
func RegisterUser(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "无效的请求参数",
		})
	}

	// 检查用户名是否已存在
	if exists := checkUserExists("username", req.Username); exists {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "用户名已被使用",
		})
	}

	// 检查邮箱是否已存在
	if exists := checkUserExists("email", req.Email); exists {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "邮箱已被注册",
		})
	}

	// 创建新用户
	newUser := model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password, // BeforeCreate 钩子会自动加密
		IsActive: true,
	}

	if err := database.DB().Create(&newUser).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "注册失败，请稍后重试",
		})
	}

	// 返回创建成功的用户信息（排除敏感字段）
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "注册成功",
		"user":    newUser.GetUserInfo(),
	})
}

// 检查用户字段是否已存在
func checkUserExists(field string, value string) bool {
	var count int64
	database.DB().Model(&model.User{}).Where(field+" = ?", value).Count(&count)
	return count > 0
}

func InitRouting(app *fiber.App) {
	app.Post("/login", Login)
	app.Post("/register", RegisterUser)
}