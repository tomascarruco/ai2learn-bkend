package routes

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"

	"github.com/tomascarruco/ai2learn-bkend/authentication"
	"github.com/tomascarruco/ai2learn-bkend/services/gcloud"
	"github.com/tomascarruco/ai2learn-bkend/services/media"
	"github.com/tomascarruco/ai2learn-bkend/web/ui/pages"
)

func SetupRouting(app *fiber.App) {
	// API
	api := app.Group("/api")

	v1 := api.Group("/v1", func(c *fiber.Ctx) error {
		c.Set("Version", "V1")
		return c.Next()
	})

	auth := v1.Group("/session")
	auth.Post("/", HandleNewSessionRequest)

	contentGeneration := v1.Group("gen/")
	contentGeneration.Route(
		"/summary",
		func(router fiber.Router) {
			router.Get("/document/:name", HandleDocumentSummary)
			router.Get("/image/:name", HandleImageSummary)
		},
		"gen_ai_summary.",
	)
	contentGeneration.Route(
		"/assessments",
		func(router fiber.Router) {
			router.Get("/tests", HandleTestRetrieval)
			router.Get("/quizz", HandleQuizzRetrieval)
		},
		"gen_ai_assessments.",
	)

	media := v1.Group("/media")
	media.Use(authentication.JwtMiddleware())
	media.Post("/setup", HandleNewUserWorkspaceCreation)

	media.Route(
		"/upload",
		func(router fiber.Router) {
			router.Post("/document", HandleNewDocumentUpload)
			router.Post("/image", HandleNewImageUpload)
		},
		"user_upload.",
	)

	// UI
	ui := app.Group("/")
	ui.Get("", func(c *fiber.Ctx) error {
		return Render(c, pages.IndexPage())
	})
}

func Render(c *fiber.Ctx, component templ.Component) error {
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
	return component.Render(c.Context(), c.Response().BodyWriter())
}

func HandleNewSessionRequest(c *fiber.Ctx) error {
	user := c.FormValue("user")

	if strings.TrimSpace(user) == "" {
		log.Warnw(
			"New session creation request WITH BAD USERNAME",
			"user",
			user,
		)

		c.Status(fiber.ErrBadRequest.Code)

		return c.JSON(
			fiber.Map{
				"reason": "bad username",
			},
		)
	}

	log.Infow("New session creation request", "user", user)

	jwt, err := authentication.CreateSessionJwt(user)
	if errors.Is(authentication.ErrFailedToCreateUserJWT, err) {
		log.Warnw(
			"Error authenticating the user",
			"user",
			user,
			"reason",
			err.Error(),
		)

		c.Status(fiber.ErrInternalServerError.Code)

		return c.JSON(
			fiber.Map{
				"reason": "Unable to authenticate user",
			},
		)
	}

	log.Infow("New session creation request SUCCESS", "user", user)
	return c.JSON(fiber.Map{"token": jwt})
}

func HandleNewDocumentUpload(c *fiber.Ctx) error {
	logger := log.WithContext(c.UserContext())
	logger.Infow("Received new PDF upload request")

	claims := authentication.ExtractJwtMClaims(c)
	subject := claims["name"].(string)

	formFile, err := c.FormFile("document")
	if err != nil {
		logger.Errorw("Error on retrieving file from request", "reason", err.Error())
		return fiber.ErrBadRequest
	}

	fileReader, err := formFile.Open()
	if err != nil {
		logger.Errorw("Failed to open file sent in the form", "reason", err.Error())
		return fiber.ErrBadRequest
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), time.Second*40)
	defer cancel()

	bucketPath := fmt.Sprintf("%s-media-store", subject)

	err = gcloud.GCloudStorage.UploadObjectToBucket(
		ctx,
		logger,
		media.NewUserFileUpRequest(
			bucketPath,
			media.InpDocumentsFolder,
			formFile.Filename,
			media.PDF,
			fileReader,
		),
	)
	if err != nil {
		logger.Errorw("Failure uploading object", "reason", err.Error())
		return fiber.ErrBadRequest
	}

	logger.Infow("Succes uploading a PDF!", "file_name", formFile.Filename)

	return c.SendStatus(fiber.StatusOK)
}

func HandleNewImageUpload(c *fiber.Ctx) error {
	logger := log.WithContext(c.UserContext())
	logger.Infow("Received new PDF upload request")

	claims := authentication.ExtractJwtMClaims(c)
	subject := claims["name"].(string)

	formFile, err := c.FormFile("image")
	if err != nil {
		logger.Errorw("Error on retrieving image from request", "reason", err.Error())
		return fiber.ErrBadRequest
	}

	fileReader, err := formFile.Open()
	if err != nil {
		logger.Errorw("Failed to open file sent in the form", "reason", err.Error())
		return fiber.ErrBadRequest
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), time.Second*40)
	defer cancel()

	bucketPath := fmt.Sprintf("%s-media-store", subject)

	err = gcloud.GCloudStorage.UploadObjectToBucket(
		ctx,
		logger,
		media.NewUserFileUpRequest(
			bucketPath,
			media.InImagesFolder,
			formFile.Filename,
			media.IMG,
			fileReader,
		),
	)
	if err != nil {
		logger.Errorw("Failure uploading object", "reason", err.Error())
		return fiber.ErrBadRequest
	}

	logger.Infow("Succes uploading image!", "file_name", formFile.Filename)

	return c.SendStatus(fiber.StatusOK)
}

func HandleNewUserWorkspaceCreation(c *fiber.Ctx) error {
	logger := log.WithContext(c.UserContext())
	logger.Infow("Creating new user workspace")

	claims := authentication.ExtractJwtMClaims(c)
	subject := claims["name"].(string)

	logger.Infow("New user workspace", "user", subject)

	foldersToCreate := append(media.InputFolders[:], media.OutputFolders...)

	if err := media.SetupUserMediaStorage(subject, logger, foldersToCreate...); err != nil {
		logger.Errorw("Failure creating a user workspace", "user_workspace", subject, "reason", err.Error())
		return fiber.ErrInternalServerError
	}

	logger.Infow("Success creating media workspace", "workspace", subject)

	return c.SendStatus(fiber.StatusOK)
}

// TODO:
func HandleImageSummary(c *fiber.Ctx) error {
	return fiber.ErrNotImplemented
}

type docSummaryRequestQuery struct {
	DocumentName string `query:"target"`
}

func HandleDocumentSummary(c *fiber.Ctx) error {
	logger := log.WithContext(c.UserContext())
	logger.Infow("Creating new user workspace")

	var routeQuery docSummaryRequestQuery

	if err := c.QueryParser(routeQuery); err != nil {
		logger.Warnw("Bad user request for doc summary", "reason", err.Error())
		return fiber.ErrBadRequest
	}

	claims := authentication.ExtractJwtMClaims(c)
	subject := claims["name"].(string)

	modelConnector, err := gcloud.NewGenAiModelConnector(subject, gcloud.PROMPT_DOCUMENT_SUMMARY)
	if err != nil {
		logger.Errorw("Could not create new model interaction", "reason", err)
	}

	generationRequest, err := modelConnector.SummarizeTexBasedContent("", "application/pdf")
	if err != nil {
		log.Errorw("Could not summarize document", "reason", err.Error())
		return fiber.ErrInternalServerError
	}

	return c.JSON(fiber.Map{
		"summary": generationRequest.GeneratedContent,
	})
}

// TODO:
func HandleQuizzRetrieval(c *fiber.Ctx) error {
	return fiber.ErrNotImplemented
}

// TODO:
func HandleTestRetrieval(c *fiber.Ctx) error {
	return fiber.ErrNotImplemented
}
