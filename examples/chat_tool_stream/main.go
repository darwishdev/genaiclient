package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/darwishdev/genaiclient"
	"github.com/darwishdev/genaiclient/pkg/genaiconfig"
	"github.com/redis/go-redis/v9"
	"google.golang.org/genai"
)

// ------------------------------------------------------
// Tool Implementations
// ------------------------------------------------------

func GetTimeTool(args map[string]interface{}) (string, error) {
	city, ok := args["city"].(string)
	if !ok || city == "" {
		return "", fmt.Errorf("missing required parameter: city")
	}
	// Mock time lookup — real version could call a timezone API.
	return fmt.Sprintf("The current time in %s is %s", city, time.Now().Format("15:04:05")), nil
}

// ------------------------------------------------------
// Tool Execution
// ------------------------------------------------------

func ExecuteTool(call *genaiconfig.FunctionCall) (*genaiconfig.ModelResponse, error) {
	switch call.Name {
	case "get_time":
		result, err := GetTimeTool(call.Args)
		if err != nil {
			return nil, err
		}
		return &genaiconfig.ModelResponse{Text: result}, nil
	default:
		return nil, fmt.Errorf("unknown tool: %s", call.Name)
	}
}

// ------------------------------------------------------
// Main
// ------------------------------------------------------

func main() {
	ctx := context.Background()

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY not set")
	}

	const (
		REDIS_HOST     = "localhost"
		REDIS_PORT     = 6379
		REDIS_PASSWORD = ""
		REDIS_DB       = 5
		USER_ID        = "stream_tool_user_001"
	)

	// --- Setup Gemini + Redis + Client ---
	geminiClient, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatalf("failed to create gemini client: %v", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:       fmt.Sprintf("%s:%d", REDIS_HOST, REDIS_PORT),
		Password:   REDIS_PASSWORD,
		DB:         REDIS_DB,
		ClientName: "genaiclient",
	})

	genaiClient, err := genaiclient.NewGenaiClient(ctx, geminiClient, redisClient, "gemini-2.5-flash-lite")
	if err != nil {
		log.Fatalf("failed to init genai client: %v", err)
	}

	// --- 1. Create Agent ---
	agentConfig := genaiconfig.AgentConfig{
		ID:                "TimeAgent",
		Persona:           "Helpful assistant that gives current time using a tool.",
		SystemInstruction: "You must always use the get_time tool to answer time-related questions.",
		DefaultModel:      "gemini-2.5-flash-lite",
		DefaultGenerationConfig: &genaiconfig.GenerationConfig{
			Tools: []*genaiconfig.Tool{
				{
					Name:        "get_time",
					Description: "Returns the current time for a given city.",
					Parameters: map[string]interface{}{
						"city": map[string]interface{}{
							"type":        "string",
							"description": "City name to get time for.",
						},
					},
				},
			},
			ToolConfig: &genaiconfig.ToolConfig{
				Mode:         genaiconfig.FunctionCallingModeAny, // 🔥 enforce tool call
				AllowedTools: []string{"get_time"},
			},
		},
	}
	agent, err := genaiClient.NewAgent(ctx, agentConfig)
	if err != nil {
		log.Fatalf("failed to create agent: %v", err)
	}

	// --- 2. Create Chat ---
	chatConfig := &genaiconfig.ChatConfig{
		ID:     "chat_stream_tool_demo_001",
		UserID: USER_ID,
		Model:  agentConfig.DefaultModel,
	}
	chat, err := agent.NewChat(ctx, chatConfig)
	if err != nil {
		log.Fatalf("failed to create chat: %v", err)
	}

	// --- 3. Send Streaming Message ---
	fmt.Println("🗣️ User: What’s the current time in Tokyo?\n")

	streamChan, err := chat.SendMessageStream(ctx, genaiconfig.Prompt{
		Text: "What's the current time in Tokyo?",
	})
	if err != nil {
		log.Fatalf("failed to start stream: %v", err)
	}

	fmt.Print("🤖 Agent (streaming): ")

	var fullText string
	var functionCall *genaiconfig.FunctionCall

	for msg := range streamChan {
		if msg.Error != nil {
			log.Fatalf("stream error: %v", msg.Error)
		}

		// Partial text chunks
		if msg.Text != "" {
			fmt.Print(msg.Text)
			fullText += msg.Text
		}

		// Function call (if streaming model returns it mid-stream)
		if msg.FunctionCall != nil {
			functionCall = msg.FunctionCall
		}
	}

	fmt.Println("\n\n✅ Stream complete.")

	// --- 4. Handle Tool Call if Model decided to use one ---
	if functionCall != nil {
		fmt.Println("🤖 Model decided to call function:", functionCall.Name)

		argsJSON, _ := json.MarshalIndent(functionCall.Args, "", "  ")
		fmt.Println("📦 Args:", string(argsJSON))

		result, err := ExecuteTool(functionCall)
		if err != nil {
			log.Fatalf("tool execution failed: %v", err)
		}

		fmt.Println("✅ Tool result:", result.Text)
	} else {
		fmt.Println("🗣️ Final streamed text:\n", fullText)
	}

	// --- 5. Show chat history from Redis ---
	history, err := chat.GetHistory(ctx)
	if err == nil {
		fmt.Println("\n💾 Chat history in Redis:")
		for _, m := range history {
			fmt.Printf("[%s] %s\n", m.Role, m.Content)
		}
	}
}
