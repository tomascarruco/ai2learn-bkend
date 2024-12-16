package media

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2/log"
	"github.com/tomascarruco/ai2learn-bkend/services/gcloud"
)

func SetupUserMediaStorage(userName string, logger log.CommonLogger) error {
	// TODO: Createa bucket for the user
	// TODO: Create adequate "imput" folders for PDFs and Images
	// TODO: Create the "output" folders for Content description/summary, image analysis and PDF quiz generation

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()

	bucketName := fmt.Sprintf("%s-media-store", userName)
	bucket, err := gcloud.GCloudStorage.CreateBucket(ctx, bucketName)
	if err != nil {
		logger.Errorw("Failed to create new workspace", "reason", err.Error())
		return err
	}

	inputFolders := []string{"textbased", "images"}
	outputFolders := []string{"file_summarys", "img_analysis", "quizzs"}

	filesSlice := append(inputFolders, outputFolders...)
	log.Debugw("Files slices", "slice", filesSlice)
	for _, folderName := range filesSlice {
		_, err := gcloud.GCloudStorage.CreateBucketFolder(bucket, folderName)
		if err != nil {
			logger.Errorw(
				"Failure creating a folder inside a bucket",
				"bucket",
				bucket.BucketName(),
				"folder",
				folderName,
				"reason",
				err.Error(),
			)
			return err
		}
	}

	return nil
}
