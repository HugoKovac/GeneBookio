package httpserver

import (
	"github.com/gofiber/fiber/v3"
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

	return app

}
