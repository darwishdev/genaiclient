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

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY not set")
	}

	const (
		REDIS_HOST     = "localhost"
		REDIS_PORT     = 6379
		REDIS_PASSWORD = ""
		REDIS_DB       = 4
		USER_ID        = "stream_user_001"
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
		ID:           "StreamTester",
		Persona:      "You are a calm and thoughtful assistant that explains your reasoning slowly.",
		DefaultModel: "gemini-2.5-flash-lite",
	}
	agent, err := genaiClient.NewAgent(ctx, agentConfig)
	if err != nil {
		log.Fatalf("failed to create agent: %v", err)
	}

	// --- 2. Create Chat ---
	chatConfig := &genaiconfig.ChatConfig{
		ID:     "chat_stream_demo_001",
		UserID: USER_ID,
		Model:  agentConfig.DefaultModel,
	}
	chat, err := agent.NewChat(ctx, chatConfig)
	if err != nil {
		log.Fatalf("failed to create chat: %v", err)
	}

	// --- 3. Send Streaming Message ---
	fmt.Println("🗣️ User: Describe how a rainbow forms in simple terms.\n")
	streamChan, err := chat.SendMessageStream(ctx, genaiconfig.Prompt{
		Text: "Describe how a rainbow forms in simple terms.",
	})
	if err != nil {
		log.Fatalf("failed to start stream: %v", err)
	}

	fmt.Print("🤖 Agent: ")

	var fullText string
	for msg := range streamChan {
		if msg.Error != nil {
			log.Fatalf("stream error: %v", msg.Error)
		}

		if msg.Text != "" {
			fmt.Print(msg.Text)
			fullText += msg.Text
		}
	}

	fmt.Println("\n\n✅ Stream complete.")
	fmt.Println("Full accumulated response:\n", fullText)

	// --- 4. Show chat history from Redis ---
	history, err := chat.GetHistory(ctx)
	if err == nil {
		fmt.Println("\n💾 Chat history in Redis:")
		for _, m := range history {
			fmt.Printf("[%s] %s\n", m.Role, m.Content)
		}
	}
}
