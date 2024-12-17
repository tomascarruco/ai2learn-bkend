package gcloud

import "github.com/gofiber/fiber/v2/log"

// Reference to the runtime CloudStorageHandler instance
var GCloudStorage *CloudStorageHandler

func CreateGCloudStorageHandler() {
	var err error
	GCloudStorage, err = NewCloudStorageHandler("ai2learn")
	if err != nil {
		log.Fatalw("could not connect to the gcloud storage", "reason", err.Error())
		return
	}
	log.Info("Successful conection to gcloud")
}
