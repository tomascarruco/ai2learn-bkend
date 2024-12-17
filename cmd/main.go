package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/tomascarruco/ai2learn-bkend/routes"
	"github.com/tomascarruco/ai2learn-bkend/services/gcloud"
)

func main() {
	app := fiber.New(fiber.Config{
		AppName: "Ai2Learn+beta",
	})

	gcloud.CreateGCloudStorageHandler()

	routes.SetupRouting(app)

	log.Fatal(app.Listen(":8080"))
}
