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
	genaiClient, err := genaiclient.NewGenaiClient(ctx, geminiClient, redisClient, "gemini-2.5-flash-lite")
	if err != nil {
		log.Fatal(err)
	}
	agentConfig := genaiconfig.AgentConfig{
		ID:           "Joke-Teller",
		Persona:      "You are a friendly joke teller. You must tell the user funny jokes your name is holly",
		DefaultModel: "gemini-2.5-flash-lite",
	}
	jokesAgent, err := genaiClient.NewAgent(ctx, agentConfig)

	if err != nil {
		log.Fatal(err)
	}

	if jokesAgent == nil {
		log.Fatal("nil agent")
	}
	prompt := &genaiconfig.Prompt{Text: "Tell Me Funny Joke"}
	response, err := jokesAgent.Generate(ctx, USER_ID, prompt)
	fmt.Println(err)
	fmt.Println(response)
}
