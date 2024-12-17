package gcloud

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/vertexai/genai"
	"github.com/gofiber/fiber/v2/log"
)

// VertexAI/Project model related constants
const (
	ModelLocation     = "us-central-1"
	ModelName         = "gemini-2.0-flash-exp"
	ModelTemperature  = 1
	ModelTopP         = 0.95
	ModelMaxOutTokens = 10_000
	ModelResponseType = "TEXT"
	ModelTimeout      = time.Minute
	ProjectName       = "ai2learn"
)

var AppGenAiClient *genai.Client

func SetupAppGenAiConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), ModelTimeout)
	defer cancel()

	newClient, err := genai.NewClient(ctx, "ai2learn", ModelLocation)
	if err != nil {
		log.Errorw("Failed to establish connection to vertexai service", "reason", err.Error())
		return err
	}

	AppGenAiClient = newClient

	return nil
}

type ContentGenerationEnabler interface {
	SumarizeContent() string
	DescribeContent() string
	QuestionsFromContent(qNum uint8) []string
	GeneralRequest(request string) string
}

type ContentGenerationResult struct {
	CreatedAt           time.Time `json:"created"`
	Prompt              string    `json:"prompt"`
	OptionalInteraction string    `json:"optional_interaction"`
	GeneratedContent    string    `json:"generated"`
}

type GenAiModelConnector struct {
	callerIdentifier string
	createdAt        time.Time
	sessionModel     *genai.GenerativeModel
	sessionChat      *genai.ChatSession
	basePrompt       string
}

func NewGenAiModelConnector(callerIdentifier string, prompts ...string) (*GenAiModelConnector, error) {
	if strings.TrimSpace(callerIdentifier) == "" {
		return nil, errors.New("Bad session caller identifier")
	}

	genModel := AppGenAiClient.GenerativeModel(ModelName)
	if genModel != nil {
		log.Errorw("Cloud not get genModel from the genai client", "reason", "Unknown internal genai error?")
		return nil, errors.New("Could not get genModel handle from genai client")
	}

	ims := &GenAiModelConnector{
		callerIdentifier: callerIdentifier,
		createdAt:        time.Now(),
		sessionModel:     genModel,
		sessionChat:      nil,
		basePrompt:       strings.Join(prompts, "\n"),
	}
	return ims, nil
}

func (s *GenAiModelConnector) CreateChatSession() {
	s.sessionChat = s.sessionModel.StartChat()
}

func (s *GenAiModelConnector) SummarizeTexBasedContent(
	bucketPath string,
	contentType string,
	otherPrompt ...string,
) (summary *ContentGenerationResult, err error) {
	content := genai.FileData{
		MIMEType: contentType,
		FileURI:  bucketPath,
	}

	prompt := s.basePrompt
	if strings.TrimSpace(prompt) == "" {
		prompt = strings.Join(otherPrompt, "\n")
	}

	geminiPrompt := genai.Text(prompt)

	ctx := context.Background()

	resp, err := s.sessionModel.GenerateContent(ctx, content, geminiPrompt)
	if err != nil {
		return nil, errors.New("No response returned from model")
	}

	if resp == nil {
		return nil, fmt.Errorf("Empty response returned by gemini")
	}

	if resp.UsageMetadata.CandidatesTokenCount < 1 {
		return nil, fmt.Errorf("No content in responses")
	}
	response_candidates := resp.Candidates
	if response_candidates == nil {
		return nil, fmt.Errorf("No response Candidates")
	}

	part := response_candidates[0].Content.Parts[0]

	return &ContentGenerationResult{
		CreatedAt:           time.Now(),
		Prompt:              prompt,
		OptionalInteraction: strings.Join(otherPrompt, "#!end_promp!t#"),
		GeneratedContent:    string(part.(genai.Text)),
	}, nil
}
