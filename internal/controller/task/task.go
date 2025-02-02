package task

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/lrypcy/aha_webserver/internal/database"
	"github.com/lrypcy/aha_webserver/internal/model"
)

// AddTask 创建新任务
// @Summary 创建新任务
// @Tags 任务管理
// @Accept json
// @Produce json
// @Param task body model.Task true "任务信息"
// @Success 201 {object} model.Task
// @Failure 400 {object} fiber.Map
// @Failure 500 {object} fiber.Map
// @Router /task [post]
func AddTask(c *fiber.Ctx) error {
	task := model.Task{}
	
	// 解析请求体
	if err := c.BodyParser(&task); err != nil {
		err_str := fmt.Sprintf("Invalid request body\n%s", err.Error())
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err_str,
		})
	}

	// 创建记录并处理错误
	result := database.DB().Create(&task)
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create task",
		})
	}

	// 返回创建成功的响应
	return c.Status(fiber.StatusCreated).JSON(task)
}

// GetTask 获取任务详情
// @Summary 获取任务详情
// @Tags 任务管理
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} model.Task
// @Failure 404 {object} fiber.Map
// @Failure 500 {object} fiber.Map
// @Router /task/{id} [get]
func GetTask(c *fiber.Ctx) error {
	id := c.Params("id")
	var task model.Task

	// 根据ID查询记录
	result := database.DB().Where("id = ?", id).First(&task)
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Task not found",
		})
	}
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	return c.JSON(task)
}

func InitRouting(app *fiber.App) {
	fmt.Println("add task routing")
	app.Post("/task", AddTask)
	app.Get("/task/:id", GetTask)
	fmt.Println("end task routing")
}