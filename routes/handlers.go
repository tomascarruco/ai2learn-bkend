package routes

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/tomascarruco/ai2learn-bkend/services/gcloud"
	"github.com/tomascarruco/ai2learn-bkend/services/media"
	"github.com/tomascarruco/ai2learn-bkend/web/ui/components"
	"github.com/tomascarruco/ai2learn-bkend/web/ui/pages"
)

func FileUploadPage(ctx *fiber.Ctx) error {
	return Render(ctx, pages.FileUploadPage())
}

func FileUploadHandler(ctx *fiber.Ctx) error {
	logger := log.WithContext(ctx.UserContext())
	logger.Infow("Received new PDF upload request")

	formFile, err := ctx.FormFile("document")
	if err != nil {
		logger.Errorw("Error on retrieving file from request", "reason", err.Error())
		return fiber.ErrBadRequest
	}

	fileReader, err := formFile.Open()
	if err != nil {
		logger.Errorw("Failed to open file sent in the form", "reason", err.Error())
		return fiber.ErrBadRequest
	}

	fileUpErr := sendToStorageBucket(ctx.UserContext(), logger, formFile.Filename, fileReader)
	if fileUpErr != nil {
		return fiber.ErrInternalServerError
	}

	logger.Infow("Succes uploading a PDF!", "file_name", formFile.Filename)

	fileMimeType := formFile.Header.Get("content-type")

	switch fileMimeType {
	case "application/pdf":
		res, err := getDocumentSummary(logger, formFile.Filename)
		if err != nil {
			log.Errorw("Error generating document summary", "reason", err.Error())
			return fiber.ErrBadRequest
		}

		reader := strings.NewReader(res.GeneratedContent)
		sendToSummaryBucket(
			ctx.UserContext(),
			logger,
			formFile.Filename,
			reader,
		)

		return Render(ctx, pages.GenerationResult(res.GeneratedContent))
	default:
		log.Errorw("Unsuported file mime-type", "mime-type", fileMimeType)
		return Render(ctx, components.NewToast("Could not generate content").Error())
	}
}

func getDocumentSummary(logger log.CommonLogger, fileName string) (*gcloud.ContentGenerationResult, error) {
	modelConnector, err := gcloud.NewGenAiModelConnector("user", gcloud.PROMPT_DOCUMENT_SUMMARY)
	if err != nil {
		logger.Errorw("Could not create new model interaction", "reason", err)
		return nil, err
	}

	location := fmt.Sprintf("gs://ai2learn_pdf__tc/textbased/%s", fileName)
	logger.Infow("new file bucket path", "path", location)

	generationRequest, err := modelConnector.SummarizeTexBasedContent(location, "application/pdf")
	if err != nil {
		log.Errorw("Could not summarize document", "reason", err.Error())
		return nil, err
	}

	return generationRequest, nil
}

func sendToSummaryBucket(ctx context.Context, logger log.CommonLogger, summaryName string, file io.Reader) error {
	taskCtx, cancel := context.WithTimeout(ctx, time.Second*40)
	defer cancel()

	bucketPath := "ai2learn_pdf__tc"

	err := gcloud.GCloudStorage.UploadObjectToBucket(
		taskCtx,
		logger,
		media.NewUserFileUpRequest(
			bucketPath,
			media.OutDocSummaryFolder,
			summaryName,
			media.Markdown,
			file,
		),
	)
	return err
}

func sendToStorageBucket(ctx context.Context, logger log.CommonLogger, fileName string, file io.Reader) error {
	taskCtx, cancel := context.WithTimeout(ctx, time.Second*40)
	defer cancel()

	bucketPath := "ai2learn_pdf__tc"

	err := gcloud.GCloudStorage.UploadObjectToBucket(
		taskCtx,
		logger,
		media.NewUserFileUpRequest(
			bucketPath,
			media.InpDocumentsFolder,
			fileName,
			media.PDF,
			file,
		),
	)
	return err
}
