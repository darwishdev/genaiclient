package genaiclient

import (
	"context"

	"github.com/darwishdev/genaiclient/app/agent"
	"github.com/darwishdev/genaiclient/pkg/genaiconfig"
	"github.com/darwishdev/genaiclient/pkg/redisclient"
	"github.com/pgvector/pgvector-go"
	"github.com/redis/go-redis/v9"
	"google.golang.org/genai"
)

// GenaiClientInterface is the main entry point for the library.
// It acts as a factory for agents and provides stateless services like embedding.
type GenaiClientInterface interface {
	NewAgent(ctx context.Context, agentConfig genaiconfig.AgentConfig) (agent.AgentInterface, error)
	GetAgent(ctx context.Context, agentID string) (agent.AgentInterface, error)
	ListAgents(ctx context.Context) ([]agent.AgentInterface, error)
	RemoveAgent(ctx context.Context, agentID string) error
	Embed(ctx context.Context, text string) (*pgvector.Vector, error)
	EmbedBulk(ctx context.Context, text []string) ([]*pgvector.Vector, error)
}

// Genaiclient is the concrete implementation of the GenaiClientInterface.
type Genaiclient struct {
	genaiClient *genai.Client
	redisClient redisclient.RedisClientInterface
}

// NewGenaiClient is the constructor for the Genaiclient.
func NewGenaiClient(ctx context.Context, genaiClient *genai.Client, redisInstance *redis.Client) (GenaiClientInterface, error) {
	redisCient := redisclient.NewRedisClient(redisInstance, false)
	return &Genaiclient{
		redisClient: redisCient,
		genaiClient: genaiClient,
	}, nil
}

// --- GenaiClientInterface Implementation ---

func (g *Genaiclient) NewAgent(ctx context.Context, agentConfig genaiconfig.AgentConfig) (agent.AgentInterface, error) {
	// 1. Persist the agent configuration using the Redis DAL.
	if err := g.redisClient.CreateAgent(ctx, agentConfig); err != nil {
		return nil, err
	}
	// 2. Create a new agent instance, injecting its dependencies.
	return agent.NewAgent(agentConfig, g.genaiClient, g.redisClient), nil
}

func (g *Genaiclient) GetAgent(ctx context.Context, agentID string) (agent.AgentInterface, error) {
	// 1. Retrieve the agent configuration from Redis.
	agentConfig, err := g.redisClient.GetAgent(ctx, agentID)
	if err != nil {
		return nil, err
	}
	// 2. Create an agent instance with the retrieved config.
	return agent.NewAgent(*agentConfig, g.genaiClient, g.redisClient), nil
}

func (g *Genaiclient) ListAgents(ctx context.Context) ([]agent.AgentInterface, error) {
	// Implementation to list agents from Redis.
	return nil, nil
}

func (g *Genaiclient) RemoveAgent(ctx context.Context, agentID string) error {
	return g.redisClient.RemoveAgent(ctx, agentID)
}

func (g *Genaiclient) Embed(ctx context.Context, text string) (*pgvector.Vector, error) {
	// TODO: Implement embedding logic.
	return nil, nil
}

func (g *Genaiclient) EmbedBulk(ctx context.Context, text []string) ([]*pgvector.Vector, error) {
	// TODO: Implement bulk embedding logic.
	return nil, nil
}

