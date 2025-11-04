package agent

import (
	"context"
	"fmt"

	"github.com/darwishdev/genaiclient/pkg/adapter"
	"github.com/darwishdev/genaiclient/pkg/genaiconfig"
	"github.com/darwishdev/genaiclient/pkg/redisclient"
	"google.golang.org/genai"
)

// -----------------------------------------------------------
// Error constants
// -----------------------------------------------------------

const (
	ErrToolNotFound          = "tool %q not found"
	ErrConvertAgentConfig    = "error converting the agent config to genai generation config: %w"
	ErrConvertPrompt         = "error converting the prompt to genai content: %w"
	ErrGeminiEmptyResponse   = "gemini returned empty response"
	ErrCreateChat            = "failed to create chat in redis: %w"
	ErrGetChat               = "failed to get chat from redis: %w"
	ErrConvertGeminiResponse = "error converting gemini response to model response: %w"
	ErrCreateOrUpdateAgent   = "failed to create or update agent in redis: %w"
	ErrGenerateContent       = "failed to generate content using model: %w"
)

// -----------------------------------------------------------
// Agent interface and implementation
// -----------------------------------------------------------

type AgentInterface interface {
	GetConfig() genaiconfig.AgentConfig
	AddTool(ctx context.Context, tool *genaiconfig.Tool) error
	RemoveTool(ctx context.Context, toolName string) error
	ListTools(ctx context.Context) []*genaiconfig.Tool
	Generate(ctx context.Context, userID string, prompt *genaiconfig.Prompt, overrideConfig ...*genaiconfig.ChatConfig) (*genaiconfig.ModelResponse, error)
	NewChat(ctx context.Context, chatConfig *genaiconfig.ChatConfig) (ChatInterface, error)
	GetChat(ctx context.Context, chatID string) (ChatInterface, error)
	ListChatsByUser(ctx context.Context, userID string) ([]*genaiconfig.ChatConfig, error)
}

type Agent struct {
	config       genaiconfig.AgentConfig
	genaiClient  *genai.Client
	redisClient  redisclient.RedisClientInterface
	defaultModel string
}

func NewAgent(config genaiconfig.AgentConfig, genaiClient *genai.Client, redisClient redisclient.RedisClientInterface, defaultModel string) AgentInterface {
	if config.DefaultGenerationConfig == nil {
		temp := float32(0.01)
		config.DefaultGenerationConfig = &genaiconfig.GenerationConfig{Temperature: &temp}
	}

	return &Agent{
		config:       config,
		genaiClient:  genaiClient,
		redisClient:  redisClient,
		defaultModel: defaultModel,
	}
}

// -----------------------------------------------------------
// AgentInterface Implementation
// -----------------------------------------------------------

func (a *Agent) GetConfig() genaiconfig.AgentConfig {
	return a.config
}

func (a *Agent) AddTool(ctx context.Context, tool *genaiconfig.Tool) error {
	a.config.DefaultGenerationConfig.Tools = append(a.config.DefaultGenerationConfig.Tools, tool)
	if err := a.redisClient.CreateAgent(ctx, a.config); err != nil {
		return fmt.Errorf(ErrCreateOrUpdateAgent, err)
	}
	return nil
}

func (a *Agent) RemoveTool(ctx context.Context, toolName string) error {
	for i, t := range a.config.DefaultGenerationConfig.Tools {
		if t.Name == toolName {
			a.config.DefaultGenerationConfig.Tools = append(a.config.DefaultGenerationConfig.Tools[:i], a.config.DefaultGenerationConfig.Tools[i+1:]...)
			if err := a.redisClient.CreateAgent(ctx, a.config); err != nil {
				return fmt.Errorf(ErrCreateOrUpdateAgent, err)
			}
			return nil
		}
	}
	return fmt.Errorf(ErrToolNotFound, toolName)
}
func mergeGenerationConfig(base, override *genaiconfig.GenerationConfig) {
	if override == nil {
		return
	}

	// Simple pointer-based values
	if override.Temperature != nil {
		base.Temperature = override.Temperature
	}
	if override.TopP != nil {
		base.TopP = override.TopP
	}
	if override.TopK != nil {
		base.TopK = override.TopK
	}

	// MaxOutputTokens (non-pointer int)
	if override.MaxOutputTokens != 0 {
		base.MaxOutputTokens = override.MaxOutputTokens
	}

	// Stop sequences (override completely if provided)
	if len(override.StopSequences) > 0 {
		base.StopSequences = override.StopSequences
	}

	// Response schema config
	if override.ResponseSchemaConfig != nil {
		base.ResponseSchemaConfig = override.ResponseSchemaConfig
	}

	// Tools (override if provided)
	if len(override.Tools) > 0 {
		base.Tools = override.Tools
	}

	// ToolConfig (override if provided)
	if override.ToolConfig != nil {
		base.ToolConfig = override.ToolConfig
	}
}
func (a *Agent) ListTools(ctx context.Context) []*genaiconfig.Tool {
	return a.config.DefaultGenerationConfig.Tools
}

func cloneGenerationConfig(src *genaiconfig.GenerationConfig) *genaiconfig.GenerationConfig {
	if src == nil {
		return &genaiconfig.GenerationConfig{}
	}
	cpy := *src
	return &cpy
}
func (a *Agent) Generate(ctx context.Context, userID string, prompt *genaiconfig.Prompt, overrideConfig ...*genaiconfig.ChatConfig) (*genaiconfig.ModelResponse, error) {
	baseGenConfig := cloneGenerationConfig(a.config.DefaultGenerationConfig)
	if len(overrideConfig) > 0 && overrideConfig[0] != nil && overrideConfig[0].GenerationConfig != nil {
		mergeGenerationConfig(baseGenConfig, overrideConfig[0].GenerationConfig)
	}
	config, err := adapter.GeminiConfigFromGenerationConfig(baseGenConfig)
	if err != nil {
		return nil, fmt.Errorf(ErrConvertAgentConfig, err)
	}

	user, err := a.redisClient.FindUserByID(ctx, userID)
	if err == nil && user != nil && len(user.Context) > 0 {
		userContext := fmt.Sprintf("User Context: %s", user.Context)
		config.SystemInstruction.Parts = append(config.SystemInstruction.Parts, &genai.Part{Text: userContext})
	}

	content, err := adapter.GeminiContentFromPrompt(prompt)
	if err != nil {
		return nil, fmt.Errorf(ErrConvertPrompt, err)
	}

	model := a.defaultModel
	if len(prompt.Model) > 0 {
		model = prompt.Model
	}

	genAiResponse, err := a.genaiClient.Models.GenerateContent(ctx, model, content, config)
	if err != nil {
		return nil, fmt.Errorf(ErrGenerateContent, err)
	}
	if genAiResponse == nil {
		return nil, fmt.Errorf(ErrGeminiEmptyResponse)
	}

	response, err := adapter.ModelResponseFromGeminiContent(genAiResponse.Candidates)
	if err != nil {
		return nil, fmt.Errorf(ErrConvertGeminiResponse, err)
	}
	return response, nil
}

func (a *Agent) NewChat(ctx context.Context, chatConfig *genaiconfig.ChatConfig) (ChatInterface, error) {
	chatConfig.AgentID = a.config.ID
	if err := a.redisClient.CreateChat(ctx, chatConfig); err != nil {
		return nil, fmt.Errorf(ErrCreateChat, err)
	}
	if chatConfig.GenerationConfig == nil {
		chatConfig.GenerationConfig = a.config.DefaultGenerationConfig
	}
	return NewChat(ctx, chatConfig, a.genaiClient, a.redisClient)
}

func (a *Agent) GetChat(ctx context.Context, chatID string) (ChatInterface, error) {
	chatConfig, err := a.redisClient.GetChat(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf(ErrGetChat, err)
	}

	if chatConfig.GenerationConfig == nil {
		chatConfig.GenerationConfig = a.config.DefaultGenerationConfig
	}
	return NewChat(ctx, chatConfig, a.genaiClient, a.redisClient)
}

func (a *Agent) ListChatsByUser(ctx context.Context, userID string) ([]*genaiconfig.ChatConfig, error) {
	return a.redisClient.ListChatsByUser(ctx, userID, a.config.ID)
}
