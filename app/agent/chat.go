package agent

import (
	"context"

	"github.com/darwishdev/genaiclient/pkg/genaiconfig"
	"github.com/darwishdev/genaiclient/pkg/redisclient"
	"google.golang.org/genai"
)

// ChatInterface defines the contract for a single, stateful chat session.
type ChatInterface interface {
	GetID() string
	GetUserID() string
	GetAgentID() string
	GetHistory(ctx context.Context) ([]genaiconfig.ChatMessage, error)
	SendMessage(ctx context.Context, prompt genaiconfig.Prompt, overrideConfig ...*genaiconfig.ChatConfig) (*genaiconfig.ModelResponse, error)
	SendMessageStream(ctx context.Context, prompt genaiconfig.Prompt, overrideConfig ...*genaiconfig.ChatConfig) (<-chan *genaiconfig.ModelResponse, error)
}

// Chat is the concrete implementation of the ChatInterface.
type Chat struct {
	id          string
	userID      string
	agentID     string
	genaiClient *genai.Client
	redisClient redisclient.RedisClientInterface
}

// NewChat is the constructor for a Chat session.
func NewChat(config genaiconfig.ChatConfig, userID string, agentID string, genaiClient *genai.Client, redisClient redisclient.RedisClientInterface) ChatInterface {
	return &Chat{
		id:          config.ID,
		userID:      userID,
		agentID:     agentID,
		genaiClient: genaiClient,
		redisClient: redisClient,
	}
}

// --- ChatInterface Implementation ---

func (c *Chat) GetID() string {
	return c.id
}

func (c *Chat) GetUserID() string {
	return c.userID
}

func (c *Chat) GetAgentID() string {
	return c.agentID
}

func (c *Chat) GetHistory(ctx context.Context) ([]genaiconfig.ChatMessage, error) {
	return c.redisClient.GetChatHistory(ctx, c.id)
}

func (c *Chat) SendMessage(ctx context.Context, prompt genaiconfig.Prompt, overrideConfig ...*genaiconfig.ChatConfig) (*genaiconfig.ModelResponse, error) {
	// 1. TODO: Construct the full prompt, including agent instructions and chat history.
	//    - Get chat history from redis: c.redisClient.GetChatHistory(ctx, c.id)
	// 2. TODO: Use the adapter to convert the prompt and config to the genai library's format.
	// 3. TODO: Call the genaiClient.
	// 4. TODO: Save the new user message and the model's response to Redis.
	//    - c.redisClient.SaveChatMessage(ctx, c.id, userMessage)
	//    - c.redisClient.SaveChatMessage(ctx, c.id, modelResponse)
	return nil, nil
}

func (c *Chat) SendMessageStream(ctx context.Context, prompt genaiconfig.Prompt, overrideConfig ...*genaiconfig.ChatConfig) (<-chan *genaiconfig.ModelResponse, error) {
	// TODO: Implement streaming logic, similar to SendMessage but with a channel.
	return nil, nil
}
