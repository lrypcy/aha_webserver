package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/lrypcy/aha_webserver/internal/controller/job"
	"github.com/lrypcy/aha_webserver/internal/controller/task"
)

func InitRouting(app *fiber.App){
	initTest(app)
	task.InitRouting(app)
	job.InitRouting(app)
}
