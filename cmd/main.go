package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/tomascarruco/ai2learn-bkend/routes"
	"github.com/tomascarruco/ai2learn-bkend/services/gcloud"
)

// TODO: Adicionar public files delivery

func main() {
	gcloud.CreateGCloudStorageHandler()
	if err := gcloud.SetupAppGenAiConnection(); err != nil {
		log.Fatalf("Error setting up gen ai connection: %+v", err)
	}

	app := fiber.New(fiber.Config{
		AppName: "Ai2Learn+beta",
	})

	app.Use(favicon.New(favicon.Config{
		File: "web/public/assets/imgs/favicon.png",
		URL:  "/favicon.ico",
	}))

	app.Static(
		"/public",
		"web/public",
		fiber.Static{
			Compress: true,
			Browse:   true,
			Download: false,
			MaxAge:   60 * 60,
		},
	)

	routes.SetupRouting(app)

	log.Fatal(app.Listen(":8080"))
}
