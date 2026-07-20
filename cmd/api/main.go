package main

import (
	"context"
	"log"

	"hkorpo/book/internal/user"
	"hkorpo/book/pkg/ent"
	fiberlogger "hkorpo/book/pkg/fiber/logger"

	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"

	_ "github.com/go-sql-driver/mysql"
)

type ConfigDB struct {
	USER          string `envconfig:"MYSQL_USER"`
	PASSWORD      string `envconfig:"MYSQL_PASSWORD"`
	ROOT_PASSWORD string `envconfig:"MYSQL_ROOT_PASSWORD"`
	DATABASE      string `envconfig:"MYSQL_DATABASE"`
	HOST          string `envconfig:"MYSQL_HOST"`
}

type Config struct {
	ConfigDB
}

func main() {
	var (
		ctx    context.Context = context.Background()
		app    *fiber.App      = fiber.New()
		config Config
	)
	err := godotenv.Load("cmd/api/.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	if err := envconfig.Process("", &config); err != nil {
		log.Fatal(err.Error())
	}

	dbClient, err := ent.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=True",
		config.ConfigDB.USER,
		config.ConfigDB.PASSWORD,
		config.ConfigDB.HOST,
		config.ConfigDB.DATABASE,
	))
	if err != nil {
		log.Fatalf("failed opening connection to mysql: %v", err)
	}

	if err := dbClient.Schema.Create(ctx); err != nil {
		log.Fatalf("database migration failed: %v", err)
	}

	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${ip} ${status} - ${latency} ${method} ${path} ${error_trace}\n",
		CustomTags: map[string]logger.LogFunc{
			"error_trace": fiberlogger.ErrorTraceLoggerTags,
		},
	}))

	userRepo := user.NewRepositoryImpl(dbClient)
	userService := user.NewService(userRepo)
	user.NewHandler(app.Group("/users"), userService)

	if err := app.Listen(":3000"); err != nil {
		log.Fatalf("fiber listen failed: %v", err)
	}
}
