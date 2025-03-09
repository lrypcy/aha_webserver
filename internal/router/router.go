package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/lrypcy/aha_webserver/internal/controller/auth"
	"github.com/lrypcy/aha_webserver/internal/controller/job"
	"github.com/lrypcy/aha_webserver/internal/controller/task"
)

func InitRouting(app *fiber.App){
	task.InitRouting(app)
	job.InitRouting(app)
	auth.InitRouting(app)
}
