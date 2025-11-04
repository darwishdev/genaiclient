package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/darwishdev/genaiclient/pkg/adapter"
	"github.com/darwishdev/genaiclient/pkg/genaiconfig"
	"github.com/darwishdev/genaiclient/pkg/redisclient"
	"github.com/rs/zerolog/log"
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
	SendToolResponse(ctx context.Context, fn genaiconfig.FunctionCall, result any) (*genaiconfig.ModelResponse, error)
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
	history, err := redisClient.GetChatHistory(ctx, config.ID)
	if err != nil {
		log.Warn().Err(err).Msg("failed to load chat history")
	}
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
		return nil, fmt.Errorf("failed to create chat session config: %w", err)
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
		return nil, fmt.Errorf("error saving the redis SaveChatMessage : %w ", err)

	}

	resp, err := c.session.SendMessage(ctx, genai.Part{Text: prompt.Text})
	if err != nil {
		return nil, fmt.Errorf("error from Gemini SendMessage : %w ", err)
	}
	if resp != nil && len(resp.Candidates) > 0 {
		candidate := resp.Candidates[0]
		if candidate.Content != nil && len(candidate.Content.Parts) > 0 && candidate.Content.Parts[0].Text != "" {
			modelText := candidate.Content.Parts[0].Text
			modelMsg := genaiconfig.ChatMessage{Role: "model", Content: modelText}
			err = c.redisClient.SaveChatMessage(ctx, c.id, modelMsg)
			if err != nil {
				log.Warn().Err(err).Msg("failed to save model message")
			}
		}
	}
	return adapter.ModelResponseFromGeminiContent(resp.Candidates)
}

func (c *Chat) SendMessageStream(ctx context.Context, prompt genaiconfig.Prompt) (<-chan *genaiconfig.ModelResponse, error) {
	out := make(chan *genaiconfig.ModelResponse)

	go func() {
		defer close(out)

		// 1. Save user message
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

			if len(resp.Candidates) == 0 {
				continue
			}
			cand := resp.Candidates[0]
			if cand.Content == nil {
				continue
			}

			for _, part := range cand.Content.Parts {
				switch {
				// --- Text chunk ---
				case part.Text != "":
					accumulated += part.Text
					out <- &genaiconfig.ModelResponse{
						Text: part.Text,
					}

				// --- Function Call chunk ---
				case part.FunctionCall != nil:
					fn := part.FunctionCall
					out <- &genaiconfig.ModelResponse{
						FunctionCall: &genaiconfig.FunctionCall{
							Name: fn.Name,
							Args: fn.Args,
						},
					}
				}
			}
		}

		// 3. After stream ends, save final accumulated model message
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

func (c *Chat) SendToolResponse(ctx context.Context, fn genaiconfig.FunctionCall, result any) (*genaiconfig.ModelResponse, error) {
	// 1. Serialize result for Gemini and for logs
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize tool result: %w", err)
	}

	// 2. Log and persist the tool result in history
	toolMsg := genaiconfig.ChatMessage{
		Role:    "tool",
		Content: fmt.Sprintf("Tool %q responded with: %s", fn.Name, string(resultJSON)),
	}
	if err := c.redisClient.SaveChatMessage(ctx, c.id, toolMsg); err != nil {
		log.Warn().Err(err).Msg("failed to save tool response message")
	}

	// 3. Send the tool output back to Gemini
	resp, err := c.session.SendMessage(ctx, genai.Part{
		Text: fmt.Sprintf(`{"tool_response": {"name": %q, "result": %s}}`, fn.Name, string(resultJSON)),
	})
	if err != nil {
		return nil, fmt.Errorf("error sending tool response to Gemini: %w", err)
	}

	// 4. Save the model’s reply (if any)
	if resp != nil && len(resp.Candidates) > 0 {
		candidate := resp.Candidates[0]
		if candidate.Content != nil && len(candidate.Content.Parts) > 0 && candidate.Content.Parts[0].Text != "" {
			modelText := candidate.Content.Parts[0].Text
			modelMsg := genaiconfig.ChatMessage{Role: "model", Content: modelText}
			if err := c.redisClient.SaveChatMessage(ctx, c.id, modelMsg); err != nil {
				log.Warn().Err(err).Msg("failed to save model reply after tool response")
			}
		}
	}

	// 5. Convert Gemini content to our model response type
	return adapter.ModelResponseFromGeminiContent(resp.Candidates)
}
