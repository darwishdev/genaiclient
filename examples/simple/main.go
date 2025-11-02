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

// Your Google API key

func main() {
	ctx := context.Background()
	apiKey := os.Getenv("GEMINI_API_KEY")
	const REDIS_PORT = 6379
	const REDIS_HOST = "localhost"
	const REDIS_DATABASE = 4
	const REDIS_PASSWORD = ""
	const USER_ID = "123"
	geminiClient, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatal(err)
	}
	redisClient := redis.NewClient(&redis.Options{
		Addr:       fmt.Sprintf("%s:%d", REDIS_HOST, REDIS_PORT),
		Password:   REDIS_PASSWORD, // no password set
		DB:         REDIS_DATABASE, // use default DB
		ClientName: "genaiclient",
	})
	genaiClient, err := genaiclient.NewGenaiClient(ctx, geminiClient, redisClient)
	if err != nil {
		log.Fatal(err)
	}
	agentConfig := genaiconfig.AgentConfig{
		ID:      "weather-reporter-v1",
		Persona: "You are a friendly weather reporter. You must use the tools provided to answer questions about the weather.",
	}
	weatherAgent, err := genaiClient.NewAgent(ctx, agentConfig)

	if err != nil {
		log.Fatal(err)
	}

	if weatherAgent == nil {
		log.Fatal("nil agent")
	}
	prompt := genaiconfig.Prompt{Text: "Tell Me Weather On Cairo"}
	response, err := weatherAgent.GenerateWithContext(ctx, USER_ID, prompt)
	fmt.Println(err)
	fmt.Println(response)
}
