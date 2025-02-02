package router

import "github.com/gofiber/fiber/v2"

func initTest(app *fiber.App) {
	// Respond with "Hello, World!" on root path, "/"
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})
	// GET http://localhost:8080/hello%20world

	// app.Get("/:value", func(c *fiber.Ctx) error {
	// 	return c.SendString("value: " + c.Params("value"))
	// 	// => Get request with value: hello world
	// })

	// GET http://localhost:3000/john
	// app.Get("/:name?", func(c *fiber.Ctx) error {
	// 	if c.Params("name") != "" {
	// 		return c.SendString("Hello " + c.Params("name"))
	// 		// => Hello john
	// 	}
	// 	return c.SendString("Where is john?")
	// })

	// GET http://localhost:3000/api/user/john
	app.Get("/api/*", func(c *fiber.Ctx) error {
		return c.SendString("API path: " + c.Params("*"))
		// => API path: user/john
	})
}