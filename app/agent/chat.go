package agent

import (
	"context"
	"fmt"

	"github.com/darwishdev/genaiclient/pkg/adapter"
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
	SendMessage(ctx context.Context, prompt genaiconfig.Prompt) (*genaiconfig.ModelResponse, error)
	SendMessageStream(ctx context.Context, prompt genaiconfig.Prompt) (<-chan *genaiconfig.ModelResponse, error)
}

// Chat is the concrete implementation of the ChatInterface.
type Chat struct {
	id      string
	userID  string
	agentID string

	genaiClient *genai.Client
	session     *genai.Chat
	redisClient redisclient.RedisClientInterface
}

// NewChat is the constructor for a Chat session.
func NewChat(ctx context.Context, config *genaiconfig.ChatConfig, genaiClient *genai.Client, redisClient redisclient.RedisClientInterface) (ChatInterface, error) {
	// 1. Load chat history (if exists)
	history, _ := redisClient.GetChatHistory(ctx, config.ID)
	var contents []*genai.Content
	for _, msg := range history {
		role := genai.RoleUser
		if msg.Role == "model" {
			role = genai.RoleModel
		}
		contents = append(contents, &genai.Content{
			Role:  role,
			Parts: []*genai.Part{{Text: msg.Content}},
		})
	}
	// 2. Create Gemini Chat session
	genAiChatConfig, err := adapter.GeminiConfigFromGenerationConfig(config.GenerationConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat session conifg: %w", err)
	}
	session, err := genaiClient.Chats.Create(ctx, config.Model, genAiChatConfig, contents)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat session: %w", err)
	}
	return &Chat{
		id:          config.ID,
		userID:      config.UserID,
		agentID:     config.AgentID,
		genaiClient: genaiClient,
		redisClient: redisClient,
		session:     session,
	}, nil
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

func (c *Chat) SendMessage(ctx context.Context, prompt genaiconfig.Prompt) (*genaiconfig.ModelResponse, error) {
	userMsg := genaiconfig.ChatMessage{Role: "user", Content: prompt.Text}
	if err := c.redisClient.SaveChatMessage(ctx, c.id, userMsg); err != nil {
		return nil, err
	}

	resp, err := c.session.SendMessage(ctx, genai.Part{Text: prompt.Text})
	if err != nil {
		return nil, fmt.Errorf("Error From Gemini Send Message : %w ", err)
	}
	if resp != nil && len(resp.Candidates) > 0 {
		modelText := resp.Candidates[0].Content.Parts[0].Text
		modelMsg := genaiconfig.ChatMessage{Role: "model", Content: modelText}
		_ = c.redisClient.SaveChatMessage(ctx, c.id, modelMsg)
	}
	return adapter.ModelResponseFromGeminiContent(resp.Candidates)
}

func (c *Chat) SendMessageStream(ctx context.Context, prompt genaiconfig.Prompt) (<-chan *genaiconfig.ModelResponse, error) {
	out := make(chan *genaiconfig.ModelResponse)

	go func() {
		defer close(out)

		// 1. Save user message first
		userMsg := genaiconfig.ChatMessage{Role: "user", Content: prompt.Text}
		if err := c.redisClient.SaveChatMessage(ctx, c.id, userMsg); err != nil {
			out <- &genaiconfig.ModelResponse{
				Error: fmt.Errorf("failed to save user message: %v", err),
			}
			return
		}

		// 2. Start streaming Gemini response
		stream := c.session.SendMessageStream(ctx, genai.Part{Text: prompt.Text})

		var accumulated string

		for resp, err := range stream {
			if err != nil {
				out <- &genaiconfig.ModelResponse{Error: err}
				return
			}

			// 3. Extract partial text (stream chunk)
			if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
				continue
			}

			for _, part := range resp.Candidates[0].Content.Parts {
				if part.Text != "" {
					accumulated += part.Text

					out <- &genaiconfig.ModelResponse{
						Text: part.Text, // send incremental piece
					}
				}
			}
		}

		// 4. Save the full model message after stream ends
		if accumulated != "" {
			modelMsg := genaiconfig.ChatMessage{
				Role:    "model",
				Content: accumulated,
			}
			_ = c.redisClient.SaveChatMessage(ctx, c.id, modelMsg)
		}
	}()

	return out, nil
}
