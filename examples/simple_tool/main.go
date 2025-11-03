package main

import (
	"context"
	"fmt"
	"os"

	"github.com/darwishdev/genaiclient"
	"github.com/darwishdev/genaiclient/pkg/genaiconfig"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"google.golang.org/genai"
)

type WeatherRequest struct {
	City string `json:"city"`
}

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
		panic(err)
	}
	redisClient := redis.NewClient(&redis.Options{
		Addr:       fmt.Sprintf("%s:%d", REDIS_HOST, REDIS_PORT),
		Password:   REDIS_PASSWORD, // no password set
		DB:         REDIS_DATABASE, // use default DB
		ClientName: "genaiclient",
	})
	genaiClient, err := genaiclient.NewGenaiClient(ctx, geminiClient, redisClient, "gemini-2.5-flash-lite")
	if err != nil {
		panic(err)
	}
	agentConfig := genaiconfig.AgentConfig{
		ID:                "Weather-Reporter",
		Persona:           "**You are a friendly and precise weather reporter named Sunny.**",
		SystemInstruction: "You are a hyper-efficient virtual assistant. Your sole purpose is to identify a user's request for weather information in a specific city and immediately invoke the get_current_weather tool. Your response MUST be ONLY the function call. Do not generate any text, explanation, or conversational response before or after the function call. If the request is NOT a specific weather query, you MUST state: 'I am only permitted to check the weather in specific cities.",
		DefaultModel:      "gemini-2.5-flash-lite",
		DefaultGenerationConfig: &genaiconfig.GenerationConfig{Tools: []*genaiconfig.Tool{
			{
				Name:        "get_current_weather",
				Description: "Retrieves the current weather, temperature, and forecast summary for a specified city and, optionally, a state/region and country. Must be used anytime the user asks for the weather.",
				RequestConfig: &genaiconfig.SchemaConfig{
					Schema: WeatherRequest{},
					// SchemaJSON: map[string]interface{}{
					// 	"type": "object", // Must be "object" for parameters container
					// 	"properties": map[string]interface{}{
					// 		"city": map[string]interface{}{
					// 			"type":        "string",
					// 			"description": "The name of the city for which to get the weather. E.g., 'London', 'Tokyo', 'San Francisco'.",
					// 		},
					// 		"unit": map[string]interface{}{
					// 			"type":        "string",
					// 			"description": "The temperature unit requested by the user. Must be 'celsius' or 'fahrenheit'. Default is 'celsius'.",
					// 			"enum":        []string{"celsius", "fahrenheit"},
					// 		},
					// 	},
					// 	"required": []string{"city"}, // 'city' is the only mandatory argument
					// },
				},
			},
		}},
	}
	weatherAgent, err := genaiClient.NewAgent(ctx, agentConfig)

	if err != nil {
		panic(err)
	}

	if weatherAgent == nil {
		panic("nil agent")
	}
	prompt := &genaiconfig.Prompt{Text: "What is the current weather and forecast for London, UK"}
	response, err := weatherAgent.Generate(ctx, USER_ID, prompt)
	fmt.Println(err)
	fmt.Println("Fuction Call")
	log.Debug().Interface("Function", response.FunctionCall)
	fmt.Println(response.FunctionCall)
	fmt.Println("Fuction Text")
	fmt.Println(response.Text)
}
