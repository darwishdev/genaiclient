package genaiclient

import (
	"context"
	"encoding/json"
	"errors" // Import errors for base error definition
	"fmt"    // Import fmt for error wrapping (%w)

	"github.com/darwishdev/genaiclient/app/agent"
	"github.com/darwishdev/genaiclient/pkg/adapter"
	"github.com/darwishdev/genaiclient/pkg/genaiconfig"
	"github.com/darwishdev/genaiclient/pkg/redisclient"
	"github.com/redis/go-redis/v9"
	"google.golang.org/genai"
)

// --- Package-level Base Errors ---

var (
	// Agent errors
	ErrCreateAgentFailed = errors.New("failed to create agent configuration in redis")
	ErrGetAgentFailed    = errors.New("failed to retrieve agent configuration from redis")
	ErrRemoveAgentFailed = errors.New("failed to remove agent configuration from redis")

	// Embedding errors
	ErrContentConversionFailed = errors.New("failed to convert prompt to gemini content")
	ErrEmbedContentFailed      = errors.New("gemini api call failed to embed content")
	ErrEmbedBulkFailed         = errors.New("failed to embed one or more contents in bulk operation")
)

// GenaiClientInterface is the main entry point for the library.
// It acts as a factory for agents and provides stateless services like embedding.
type GenaiClientInterface interface {
	NewAgent(ctx context.Context, agentConfig genaiconfig.AgentConfig) (agent.AgentInterface, error)
	GetAgent(ctx context.Context, agentID string) (agent.AgentInterface, error)
	ListAgents(ctx context.Context) ([]*genaiconfig.AgentConfig, error)
	RemoveAgent(ctx context.Context, agentID string) error
	Embed(ctx context.Context, text string, options ...*EmbedOptions) ([]float32, error)
	EmbedBulk(ctx context.Context, text []string, options ...*EmbedOptions) ([][]float32, error)
	BuildGeminiTools(tools []*genaiconfig.Tool) ([]*genai.Tool, error)
	BuildGeminiTool(tool *genaiconfig.Tool) (*genai.Tool, error)
}

// Genaiclient is the concrete implementation of the GenaiClientInterface.
type Genaiclient struct {
	genaiClient           *genai.Client
	defaultModel          string
	defaultEmbeddingModel string
	redisClient           redisclient.RedisClientInterface
}

// NewGenaiClient is the constructor for the Genaiclient.
func NewGenaiClient(ctx context.Context, genaiClient *genai.Client, redisInstance *redis.Client, defaultModel string, defaultEmbeddingModel string) (GenaiClientInterface, error) {
	redisCient := redisclient.NewRedisClient(redisInstance, false)
	return &Genaiclient{
		redisClient:           redisCient,
		defaultModel:          defaultModel,
		defaultEmbeddingModel: defaultEmbeddingModel,
		genaiClient:           genaiClient,
	}, nil
}

// --- GenaiClientInterface Implementation ---

func (g *Genaiclient) NewAgent(ctx context.Context, agentConfig genaiconfig.AgentConfig) (agent.AgentInterface, error) {
	// 1. Persist the agent configuration using the Redis DAL.
	if agentConfig.DefaultModel == "" {
		agentConfig.DefaultModel = g.defaultModel
	}
	if err := g.redisClient.CreateAgent(ctx, agentConfig); err != nil {
		// Error wrapping added
		return nil, fmt.Errorf("%w: %w", ErrCreateAgentFailed, err)
	}
	// 2. Create a new agent instance, injecting its dependencies.
	return agent.NewAgent(agentConfig, g.genaiClient, g.redisClient, agentConfig.DefaultModel), nil
}

func (g *Genaiclient) GetAgent(ctx context.Context, agentID string) (agent.AgentInterface, error) {
	// 1. Retrieve the agent configuration from Redis.
	agentConfig, err := g.redisClient.GetAgent(ctx, agentID)
	if err != nil {
		// Error wrapping added
		return nil, fmt.Errorf("%w for agentID %s: %w", ErrGetAgentFailed, agentID, err)
	}
	// 2. Create an agent instance with the retrieved config.
	return agent.NewAgent(*agentConfig, g.genaiClient, g.redisClient, g.defaultModel), nil
}

func (g *Genaiclient) ListAgents(ctx context.Context) ([]*genaiconfig.AgentConfig, error) {
	return g.redisClient.ListAgents(ctx)
}

func (g *Genaiclient) RemoveAgent(ctx context.Context, agentID string) error {
	err := g.redisClient.RemoveAgent(ctx, agentID)
	if err != nil {
		// Error wrapping added
		return fmt.Errorf("%w for agentID %s: %w", ErrRemoveAgentFailed, agentID, err)
	}
	return nil
}

type EmbedOptions struct {
	Model      string
	Dimensions int32
}

func (g *Genaiclient) Embed(ctx context.Context, text string, options ...*EmbedOptions) ([]float32, error) {
	content, err := adapter.GeminiContentFromPrompt(&genaiconfig.Prompt{Text: text})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrContentConversionFailed, err)
	}
	embeddingModel := g.defaultEmbeddingModel
	var genaiConfig *genai.EmbedContentConfig

	// Check if options were provided and are non-nil
	if len(options) > 0 && options[0] != nil {
		opts := options[0]

		// Override model if specified
		if opts.Model != "" {
			embeddingModel = opts.Model
		}

		// Only set the genaiConfig if dimensions are provided and valid (e.g., > 0)
		if opts.Dimensions > 0 {
			dim := opts.Dimensions // Store value in a variable to get its address

			genaiConfig = &genai.EmbedContentConfig{
				// Set the specific dimension size
				OutputDimensionality: &dim,
				// Set the TaskType (highly recommended for retrieval)
				TaskType: "RETRIEVAL_DOCUMENT",
			}
		}
	}
	embed, err := g.genaiClient.Models.EmbedContent(ctx, embeddingModel, content, genaiConfig)
	if err != nil {
		return nil, fmt.Errorf("%w with model %s: %w", ErrEmbedContentFailed, g.defaultModel, err)
	}
	return embed.Embeddings[0].Values, nil // Returns []float32, not [][]float32
}

const maxErrorTextLength = 250

func truncateString(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
func (g *Genaiclient) EmbedBulk(ctx context.Context, text []string, options ...*EmbedOptions) ([][]float32, error) {
	response := make([][]float32, len(text))

	for index, v := range text {
		res, err := g.Embed(ctx, v, options...)
		if err != nil {
			truncatedV := truncateString(v, maxErrorTextLength)
			return nil, fmt.Errorf("%w at index %d: %w , (value: '%s') ", ErrEmbedBulkFailed, index, err, truncatedV)
		}
		response[index] = res
	}
	return response, nil
}

func (g *Genaiclient) BuildGeminiTools(tools []*genaiconfig.Tool) ([]*genai.Tool, error) {
	return adapter.BuildGeminiTools(tools)
}
func (g *Genaiclient) BuildGeminiTool(tool *genaiconfig.Tool) (*genai.Tool, error) {
	return adapter.BuildGeminiTool(tool)
}

func BuildSchemaFromStruct[T interface{}](instance T) *genai.Schema {
	return adapter.BuildSchemaFromStruct(instance)
}

// GenerateStructured generates content with a structured response based on the generic type T.
// It automatically sets the response schema to match the structure of T.
func GenerateStructured[T any](ctx context.Context, agent agent.AgentInterface, userID string, prompt *genaiconfig.Prompt, overrideConfig ...*genaiconfig.ChatConfig) (*T, error) {
	// Create an instance of T to build the schema
	var instance T
	schema := adapter.BuildSchemaFromStruct(instance)
	var configToUse *genaiconfig.ChatConfig
	if len(overrideConfig) > 0 && overrideConfig[0] != nil {
		configToUse = overrideConfig[0]
		if configToUse.GenerationConfig == nil {
			configToUse.GenerationConfig = &genaiconfig.GenerationConfig{}
		}
	} else {
		configToUse = &genaiconfig.ChatConfig{
			GenerationConfig: &genaiconfig.GenerationConfig{},
		}
	}

	// Set the response schema
	configToUse.GenerationConfig.ResponseSchemaConfig = &genaiconfig.SchemaConfig{
		Schema: schema,
	}

	// Call the base Generate function
	response, err := agent.Generate(ctx, userID, prompt, configToUse)
	if err != nil {
		return nil, err
	}

	// Check if response has content
	if response == nil || len(response.Text) == 0 {
		return nil, fmt.Errorf("empty response")
	}

	// Unmarshal the response into the target type
	var result T
	if err := json.Unmarshal([]byte(response.Text), &result); err != nil {
		return nil, fmt.Errorf("error unmarshaling json :%w", err)
	}

	return &result, nil
}

func SendStructured[T any](
	ctx context.Context,
	chat agent.ChatInterface,
	prompt *genaiconfig.Prompt,
	overrideConfig ...*genaiconfig.ChatConfig,
) (*T, error) {

	// Create an instance of T
	var instance T
	schema := adapter.BuildSchemaFromStruct(instance)

	// Build final config
	var configToUse *genaiconfig.ChatConfig
	if len(overrideConfig) > 0 && overrideConfig[0] != nil {
		configToUse = overrideConfig[0]
		if configToUse.GenerationConfig == nil {
			configToUse.GenerationConfig = &genaiconfig.GenerationConfig{}
		}
	} else {
		configToUse = &genaiconfig.ChatConfig{
			GenerationConfig: &genaiconfig.GenerationConfig{},
		}
	}

	// Inject schema
	configToUse.GenerationConfig.ResponseSchemaConfig = &genaiconfig.SchemaConfig{
		Schema: schema,
	}

	// Send the message
	resp, err := chat.SendMessage(ctx, *prompt, configToUse)
	if err != nil {
		return nil, err
	}

	if resp == nil || len(resp.Text) == 0 {
		return nil, fmt.Errorf("empty response")
	}

	// Unmarshal into T
	var result T
	if err := json.Unmarshal([]byte(resp.Text), &result); err != nil {
		return nil, fmt.Errorf("error unmarshaling json: %w", err)
	}

	return &result, nil
}
