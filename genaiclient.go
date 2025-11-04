package genaiclient

import (
	"context"
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
	Embed(ctx context.Context, text string, model ...string) ([][]float32, error)
	EmbedBulk(ctx context.Context, text []string, model ...string) ([][][]float32, error)
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

func (g *Genaiclient) Embed(ctx context.Context, text string, model ...string) ([][]float32, error) {
	content, err := adapter.GeminiContentFromPrompt(&genaiconfig.Prompt{Text: text})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrContentConversionFailed, err)
	}
	embeddingModel := g.defaultEmbeddingModel
	if len(model) > 0 {
		embeddingModel = model[0]
	}
	embed, err := g.genaiClient.Models.EmbedContent(ctx, embeddingModel, content, nil)
	if err != nil {
		return nil, fmt.Errorf("%w with model %s: %w", ErrEmbedContentFailed, g.defaultModel, err)
	}
	response := make([][]float32, len(embed.Embeddings))
	for index, embedding := range embed.Embeddings {
		response[index] = embedding.Values
	}
	return response, nil
}

func (g *Genaiclient) EmbedBulk(ctx context.Context, text []string, model ...string) ([][][]float32, error) {
	response := make([][][]float32, len(text))
	for index, v := range text {
		embeddingModel := g.defaultEmbeddingModel
		if len(model) > 0 {
			embeddingModel = model[0]
		}
		res, err := g.Embed(ctx, v, embeddingModel)
		if err != nil {
			// Error wrapping added, including the index of the failed item
			return nil, fmt.Errorf("%w at index %d: %w", ErrEmbedBulkFailed, index, err)
		}
		response[index] = res
	}
	return response, nil
}
