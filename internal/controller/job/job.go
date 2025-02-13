package job

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/lrypcy/aha_webserver/internal/database"
	"github.com/lrypcy/aha_webserver/internal/model"
)

// AddJob 创建新任务
// @Summary 创建新Job
// @Tags Job管理
// @Accept json
// @Produce json
// @Param job body model.Job true "信息"
// @Success 201 {object} model.Job
// @Failure 400 {object} fiber.Map
// @Failure 500 {object} fiber.Map
// @Router /job [post]
func AddJob(c *fiber.Ctx) error {
	job := model.Job{}
	
	// 解析请求体
	if err := c.BodyParser(&job); err != nil {
		err_str := fmt.Sprintf("Invalid request body\n%s", err.Error())
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err_str,
		})
	}

	// 创建记录并处理错误
	result := database.DB().Create(&job)
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create job",
		})
	}

	// 返回创建成功的响应
	return c.Status(fiber.StatusCreated).JSON(job)
}

// GetJob 获取任务详情
// @Summary 获取任务详情
// @Tags 任务管理
// @Produce json
// @Param id path string true "JobID"
// @Success 200 {object} model.Job
// @Failure 404 {object} fiber.Map
// @Failure 500 {object} fiber.Map
// @Router /job/{id} [get]
func GetJob(c *fiber.Ctx) error {
	id := c.Params("id")
	var job model.Job

	// 根据ID查询记录
	result := database.DB().Where("id = ?", id).First(&job)
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Job not found",
		})
	}
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
		})
	}

	return c.JSON(job)
}

func InitRouting(app *fiber.App) {
	fmt.Println("add job routing")
	app.Post("/job", AddJob)
	app.Get("/job/:id", GetJob)
	fmt.Println("end job routing")
}