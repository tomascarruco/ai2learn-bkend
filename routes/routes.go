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
	"github.com/tomascarruco/ai2learn-bkend/web/ui/components"
	"github.com/tomascarruco/ai2learn-bkend/web/ui/pages"
	"github.com/tomascarruco/ai2learn-bkend/web/ui/uictx"
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

	mediaRoutes := v1.Group("/media")
	mediaRoutes.Use(authentication.JwtMiddleware())
	mediaRoutes.Post("/setup", HandleNewUserWorkspaceCreation)

	mediaRoutes.Route(
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
		var routes []components.LinkProps

		cookie := c.Cookies("session")
		if cookie == "" {
			routes = []components.LinkProps{
				{
					Url:      "/session",
					Name:     "New Session",
					Disabled: false,
				},
			}
		} else {
			routes = []components.LinkProps{
				{
					Url:      "/workspace",
					Name:     "Área de Trabalho",
					Disabled: false,
				},
			}
		}

		c.Locals(
			uictx.NavOptionsKey,
			routes,
		)
		return Render(c, pages.IndexPage())
	})

	ui.Get("/session", func(c *fiber.Ctx) error {
		return Render(c, pages.SessionsPage())
	})

	ui.Route(
		"/workspace",
		func(router fiber.Router) {
			router.Get("", authentication.JwtMiddleware(), func(c *fiber.Ctx) error {
				log.Debugw("TOKEN", "token", c.Cookies("session"))
				_, err := authentication.ExtractJwtMClaims(c)
				if err != nil {
					log.Errorw("Error on parssing jwt", "reason", err.Error())
					return fiber.ErrBadRequest
				}

				foldersToCreate := append(media.InputFolders[:], media.OutputFolders...)
				folders := make([]components.FolderProps, len(foldersToCreate))

				for i, folder := range media.InputFolders {
					folders[i] = components.FolderProps{
						Name:      folder,
						Categorie: "input",
						FileCount: 0,
					}
				}

				for i, folder := range media.OutputFolders {
					folders[i] = components.FolderProps{
						Name:      folder,
						Categorie: "output",
						FileCount: 0,
					}
				}

				if err == nil {
					return Render(c, pages.WorkspaceExists(folders))
				} else {
					return Render(c, pages.Workspace())
				}
			})

			router.Get("/create", func(c *fiber.Ctx) error {
				return Render(c, pages.WorkspaceCreating())
			})

			router.Get("/create", func(c *fiber.Ctx) error {
				folders := []components.FolderProps{
					{
						Name:      "Documentos",
						Categorie: "pdfs",
						FileCount: 4,
					},
				}
				return Render(c, pages.WorkspaceCreated(folders))
			})
		},
		"workspace.",
	)
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
		return Render(c, components.NewToast("Nome inválido!").Error())
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
		return Render(c, components.NewToast("Erro inesperado, tenta mais tarde!").Error())
	}

	log.Infow("New session creation request SUCCESS", "user", user, "jwt", jwt)
	c.Cookie(&fiber.Cookie{
		Name:        "session",
		Value:       jwt,
		MaxAge:      60 * 60 * 24,
		Expires:     time.Now().Add(time.Hour * 24),
		HTTPOnly:    true,
		SameSite:    "lax",
		SessionOnly: true,
	})
	return Render(c, pages.SessionSuccess())
}

func HandleNewDocumentUpload(c *fiber.Ctx) error {
	logger := log.WithContext(c.UserContext())
	logger.Infow("Received new PDF upload request")

	claims, err := authentication.ExtractJwtMClaims(c)
	if err != nil {
		logger.Errorw("Error on parssing jwt", "reason", err.Error())
		return fiber.ErrBadRequest
	}
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

	claims, err := authentication.ExtractJwtMClaims(c)
	if err != nil {
		logger.Errorw("Error on parssing jwt", "reason", err.Error())
		return fiber.ErrBadRequest
	}
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

	claims, err := authentication.ExtractJwtMClaims(c)
	if err != nil {
		logger.Errorw("Error on parssing jwt", "reason", err.Error())
		return fiber.ErrTeapot
	}
	subject := claims["name"].(string)
	subject = strings.ToLower(subject)

	logger.Infow("New user workspace", "user", subject)

	foldersToCreate := append(media.InputFolders[:], media.OutputFolders...)

	if err := media.SetupUserMediaStorage(subject, logger, foldersToCreate...); err != nil {
		logger.Errorw("Failure creating a user workspace", "user_workspace", subject, "reason", err.Error())

		c.Status(fiber.StatusInternalServerError)
		return Render(c, components.NewToast("Ocurreu algo inesperado, tente mais tarde.").Error())
	}

	logger.Infow("Success creating media workspace", "workspace", subject)

	folders := make([]components.FolderProps, len(foldersToCreate))

	for i, folder := range media.InputFolders {
		folders[i] = components.FolderProps{
			Name:      folder,
			Categorie: "input",
			FileCount: 0,
		}
	}

	for i, folder := range media.OutputFolders {
		folders[i] = components.FolderProps{
			Name:      folder,
			Categorie: "output",
			FileCount: 0,
		}
	}

	return Render(c, pages.WorkspaceCreated(folders))
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

	claims, err := authentication.ExtractJwtMClaims(c)
	if err != nil {
		logger.Errorw("Error on parssing jwt", "reason", err.Error())
		return fiber.ErrBadRequest
	}
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
