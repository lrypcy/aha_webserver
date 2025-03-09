package main

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	_ "github.com/lrypcy/aha_webserver/api"
	"github.com/lrypcy/aha_webserver/internal/database"
	"github.com/lrypcy/aha_webserver/internal/middleware/jwt"
	"github.com/lrypcy/aha_webserver/internal/router"
	"github.com/spf13/viper"
	fiberSwagger "github.com/swaggo/fiber-swagger"
)

// @title aha_webserver API
// @version 1.0
// @description This is a sample server Petstore server.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name BSD 3-Clause License
// @license.url https://opensource.org/license/bsd-3-clause

// @host petstore.swagger.io
// @BasePath /v2
func main() {
	viper.SetConfigFile("configs/config.yml")
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		fmt.Println(err)
		return
	}
	viper.SetDefault("database.type", "sqlite")
	viper.SetDefault("database.dbname", "sqlite.db")
	database.InitDB()
	jwt.Init(
		viper.GetString("jwt.secret"),
		viper.GetDuration("jwt.expiration"),
	)

	app := fiber.New()
	app.Get("/swagger/*", fiberSwagger.WrapHandler)
	router.InitRouting(app)
	app.Listen(":80")
}
