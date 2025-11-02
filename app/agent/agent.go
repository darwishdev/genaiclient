package agent

import (
	"context"
	"fmt"

	"github.com/darwishdev/genaiclient/pkg/genaiconfig"
	"github.com/darwishdev/genaiclient/pkg/redisclient"
	"google.golang.org/genai"
)

// AgentInterface defines the contract for an AI agent's capabilities.
type AgentInterface interface {
	GetConfig() genaiconfig.AgentConfig
	AddTool(ctx context.Context, goFunc interface{}) error
	RemoveTool(ctx context.Context, toolName string) error
	ListTools(ctx context.Context) ([]string, error)
	GenerateWithContext(ctx context.Context, userID string, prompt genaiconfig.Prompt, overrideConfig ...*genaiconfig.ChatConfig) (*genaiconfig.ModelResponse, error)
	NewChat(ctx context.Context, userID string, chatConfig genaiconfig.ChatConfig) (ChatInterface, error)
	GetChat(ctx context.Context, userID string, chatID string) (ChatInterface, error)
	ListChatsByUser(ctx context.Context, userID string) ([]ChatInterface, error)
}

// Agent is the concrete implementation of the AgentInterface.
type Agent struct {
	config      genaiconfig.AgentConfig
	genaiClient *genai.Client
	redisClient redisclient.RedisClientInterface
}

// NewAgent is the constructor for the Agent.
func NewAgent(config genaiconfig.AgentConfig, genaiClient *genai.Client, redisClient redisclient.RedisClientInterface) AgentInterface {
	return &Agent{
		config:      config,
		genaiClient: genaiClient,
		redisClient: redisClient,
	}
}

// --- AgentInterface Implementation ---

func (a *Agent) GetConfig() genaiconfig.AgentConfig {
	return a.config
}

func (a *Agent) AddTool(ctx context.Context, goFunc interface{}) error {
	// TODO: 1. Convert goFunc to a Tool struct using the adapter.
	// TODO: 2. Append the new tool to a.config.Tools.
	// TODO: 3. Call a.redisClient.UpdateAgent(ctx, a.config) to persist the change.
	return nil
}

func (a *Agent) RemoveTool(ctx context.Context, toolName string) error {
	// TODO: 1. Find and remove the tool from a.config.Tools.
	// TODO: 2. Call a.redisClient.UpdateAgent(ctx, a.config) to persist the change.
	return nil
}

func (a *Agent) ListTools(ctx context.Context) ([]string, error) {
	// Implementation to list tools from a.config.Tools.
	return nil, nil
}

func (a *Agent) GenerateWithContext(ctx context.Context, userID string, prompt genaiconfig.Prompt, overrideConfig ...*genaiconfig.ChatConfig) (*genaiconfig.ModelResponse, error) {
	// 1. Fetch user context for personalization.
	user, err := a.redisClient.FindUserByID(ctx, userID)
	if err != nil {
		// Handle case where user is not found, maybe proceed without context.
	}

	// 2. Prepend user context to the prompt if it exists.
	if user != nil && user.Context != "" {
		originalPrompt := prompt.Text
		prompt.Text = fmt.Sprintf("User Context: %s\n\n---\n\n%s", user.Context, originalPrompt)
	}

	// 3. TODO: Call the genaiClient with the combined prompt and agent instructions.
	return nil, nil
}

func (a *Agent) NewChat(ctx context.Context, userID string, chatConfig genaiconfig.ChatConfig) (ChatInterface, error) {
	// 1. TODO: Create a new chat record in Redis if needed.
	// 2. Return a new chat instance, injecting dependencies.
	return NewChat(chatConfig, userID, a.config.ID, a.genaiClient, a.redisClient), nil
}

func (a *Agent) GetChat(ctx context.Context, userID string, chatID string) (ChatInterface, error) {
	// TODO: 1. Verify from Redis that this chatID belongs to this userID.
	// TODO: 2. Reconstruct the chat instance.
	return nil, nil
}

func (a *Agent) ListChatsByUser(ctx context.Context, userID string) ([]ChatInterface, error) {
	// TODO: Implement logic to list chats for a user from Redis.
	return nil, nil
}
