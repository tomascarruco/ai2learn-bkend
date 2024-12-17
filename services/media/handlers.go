package media

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2/log"
	gcloud "github.com/tomascarruco/ai2learn-bkend/services/gcloud"
)

func SetupUserMediaStorage(userName string, logger log.CommonLogger, folders ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()

	bucketName := fmt.Sprintf("%s-media-store", userName)
	bucket, err := gcloud.GCloudStorage.CreateBucket(ctx, bucketName)
	if err != nil {
		logger.Errorw("Failed to create new workspace", "reason", err.Error())
		return err
	}

	log.Debugw("Files slices", "slice", folders)
	for _, folderName := range folders {
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

func UploadImage(fileName string, content []byte, contentType AcceptedObjContentType) error {
	return errors.New("Unimplemented")
}
