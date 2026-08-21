package httpserver

import (
	"github.com/gofiber/contrib/v3/swaggerui"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"

	fiberlogger "hkorpo/book/pkg/fiber/logger"
)

func Init() *fiber.App {
	var app *fiber.App = fiber.New()

	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${ip} ${status} - ${latency} ${method} ${path} ${error_trace}\n",
		CustomTags: map[string]logger.LogFunc{
			"error_trace": fiberlogger.ErrorTraceLoggerTags,
		},
	}))

	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost", "https://localhost", "capacitor://localhost"},
		AllowHeaders: []string{"Content-Type", "Authorization"},
	}))

	app.Use(swaggerui.New(swaggerui.Config{
		BasePath: "/",
		FilePath: "./docs/swagger.json",
		Path:     "docs",
		Title:    "Book Narration API",
	}))

	return app
}
