package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/darwishdev/genaiclient"
	"github.com/darwishdev/genaiclient/pkg/genaiconfig"
	"github.com/redis/go-redis/v9"
	"google.golang.org/genai"
)

func GetWeatherTool(args map[string]interface{}) (string, error) {
	city, ok := args["city"].(string)
	if !ok || city == "" {
		return "", fmt.Errorf("missing required parameter: city")
	}
	// Simulated API call
	return fmt.Sprintf("The weather in %s is 12°C and sunny ☀️", city), nil
}

// ------------------------------------------------------
// Mock Agent Simulation
// ------------------------------------------------------

// ------------------------------------------------------
// Tool Execution Handler
// ------------------------------------------------------

func ExecuteTool(call *genaiconfig.FunctionCall) (*genaiconfig.ModelResponse, error) {
	switch call.Name {
	case "get_weather":
		result, err := GetWeatherTool(call.Args)
		if err != nil {
			return nil, err
		}
		return &genaiconfig.ModelResponse{Text: result}, nil
	default:
		return nil, fmt.Errorf("unknown tool: %s", call.Name)
	}
}

type WeatherFindParams struct {
	City string
}

func main() {
	ctx := context.Background()

	// --- Setup ---
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY not set")
	}

	const (
		REDIS_HOST     = "localhost"
		REDIS_PORT     = 6379
		REDIS_PASSWORD = ""
		REDIS_DB       = 4
		USER_ID        = "user123"
	)

	// --- Create Gemini + Redis clients ---
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

	// --- 1. Create new Agent ---
	agentConfig := genaiconfig.AgentConfig{
		ID:                "ChatTester",
		Persona:           "You Name Is Weather Getter You Are A special agent for getting weather for a specif city",
		SystemInstruction: "You are weather find who can infer the ciry from user prompt and make a function call to call the get_weather tool like if the user sent to tell me the weather on cairo we must call the tool with params set to cairo as city .. also you must not answer any side questions",
		DefaultModel:      "gemini-2.5-flash-lite",
		DefaultGenerationConfig: &genaiconfig.GenerationConfig{
			ToolConfig: &genaiconfig.ToolConfig{
				Mode: genaiconfig.FunctionCallingModeAny,
			},
			Tools: []*genaiconfig.Tool{
				{
					Name:        "get_weather",
					Description: "Retrieve the current weather for a given city.",
					Parameters: map[string]interface{}{
						"params": &WeatherFindParams{},
					},
				},
			},
		},)
	}
	agent, err := genaiClient.NewAgent(ctx, agentConfig)
	if err != nil {
		log.Fatalf("failed to create agent: %v", err)
	}

	resp, err := agent.Generate(ctx, USER_ID, &genaiconfig.Prompt{Text: "Find Wather On the city that named Cairo inside Egypt"})
	// // --- 2. Create new Chat ---
	// chatConfig := &genaiconfig.ChatConfig{
	// 	ID:     "chat-001",
	// 	UserID: USER_ID,
	// 	Model:  agentConfig.DefaultModel,
	// }
	// chat, err := agent.NewChat(ctx, chatConfig)
	// if err != nil {
	// 	log.Fatalf("failed to create chat: %v", err)
	// }
	//
	// // --- 3. Send messages sequentially ---
	// // fmt.Println("\n🗣️ User: Can you tell me current weather on cairo?")
	// resp2, err := chat.SendMessage(ctx, genaiconfig.Prompt{Text: "tell me the weather on cairo"})
	if err != nil {
		log.Fatalf("send message 2 failed: %v", err)
	}
	fmt.Println("🤖 Agent:", resp)
	// // --- 4. Optional: show history from Redis ---
	// history, _ := chat.GetHistory(ctx)
	// fmt.Println("\n💾 Chat history in Redis:")
	//
	//	for _, msg := range history {
	//		fmt.Printf("[%s] %s\n", msg.Role, msg.Content)
	//	}
}
