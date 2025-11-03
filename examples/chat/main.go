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
		ID:                      "ChatTester",
		Persona:                 "You are a helpful assistant that answers concisely.",
		DefaultModel:            "gemini-2.5-flash-lite",
		DefaultGenerationConfig: &genaiconfig.GenerationConfig{},
	}
	agent, err := genaiClient.NewAgent(ctx, agentConfig)
	if err != nil {
		log.Fatalf("failed to create agent: %v", err)
	}

	// --- 2. Create new Chat ---
	chatConfig := &genaiconfig.ChatConfig{
		ID:     "chat-001",
		UserID: USER_ID,
		Model:  agentConfig.DefaultModel,
	}
	chat, err := agent.NewChat(ctx, chatConfig)
	if err != nil {
		log.Fatalf("failed to create chat: %v", err)
	}

	// --- 3. Send messages sequentially ---
	fmt.Println("🗣️ User: Hi there!")
	resp1, err := chat.SendMessage(ctx, genaiconfig.Prompt{Text: "Hi there!"})
	if err != nil {
		log.Fatalf("send message 1 failed: %v", err)
	}
	fmt.Println("🤖 Agent:", resp1.Text)

	fmt.Println("\n🗣️ User: Can you tell me a quick fact about space?")
	resp2, err := chat.SendMessage(ctx, genaiconfig.Prompt{Text: "Can you tell me a quick fact about space?"})
	if err != nil {
		log.Fatalf("send message 2 failed: %v", err)
	}
	fmt.Println("🤖 Agent:", resp2.Text)

	// --- 4. Optional: show history from Redis ---
	history, _ := chat.GetHistory(ctx)
	fmt.Println("\n💾 Chat history in Redis:")
	for _, msg := range history {
		fmt.Printf("[%s] %s\n", msg.Role, msg.Content)
	}
}
