package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/darwishdev/genaiclient"
	"github.com/darwishdev/genaiclient/pkg/genaiconfig"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/genai"
)

// ------------------------------------------------------
// Tool Implementations
// ------------------------------------------------------

type GetTimeRequest struct {
	City string `json:"city"`
}

func GetTimeTool(args map[string]interface{}) (string, error) {
	city, ok := args["city"].(string)
	if !ok || city == "" {
		return "", fmt.Errorf("missing required parameter: city")
	}
	time.Sleep(500 * time.Millisecond) // simulate delay
	now := time.Now().Format("15:04:05")
	return fmt.Sprintf("🕒 The current time in %s is %s", city, now), nil
}

// Centralized tool dispatcher
func ExecuteTool(call *genaiconfig.FunctionCall) (*genaiconfig.ModelResponse, error) {
	log.Info().
		Str("function", call.Name).
		Interface("args", call.Args).
		Msg("Tool execution started")

	switch call.Name {
	case "get_time":
		result, err := GetTimeTool(call.Args)
		if err != nil {
			log.Error().Err(err).Msg("Tool execution failed")
			return nil, err
		}
		log.Info().
			Str("function", call.Name).
			Str("result", result).
			Msg("Tool execution completed")
		return &genaiconfig.ModelResponse{Text: result}, nil
	default:
		err := fmt.Errorf("unknown tool: %s", call.Name)
		log.Error().Err(err).Msg("Invalid tool name")
		return nil, err
	}
}

// ------------------------------------------------------
// CLI Chat with synchronous SendMessage + Logging
// ------------------------------------------------------

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Configure zerolog
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.Kitchen})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal().Msg("GEMINI_API_KEY not set")
	}

	const (
		REDIS_HOST = "localhost"
		REDIS_PORT = 6379
		REDIS_DB   = 7
		USER_ID    = "cli_logger_user_001"
	)

	// Handle Ctrl+C gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n👋 Exiting chat...")
		cancel()
		os.Exit(0)
	}()

	// --- Initialize Clients ---
	geminiClient, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create Gemini client")
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%d", REDIS_HOST, REDIS_PORT),
		DB:   REDIS_DB,
	})

	genaiClient, err := genaiclient.NewGenaiClient(ctx, geminiClient, redisClient, "gemini-2.5-flash-lite")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to init GenAI client")
	}

	// --- Agent Configuration ---
	agentConfig := genaiconfig.AgentConfig{
		ID:                "CLILoggerAgent",
		Persona:           "A helpful assistant that uses tools always and you must infer the tool params from the user prompt for example when i ask for the time on new york you should infer the city as new york.",
		SystemInstruction: "You can call the get_time tool when asked for time in any city.",
		DefaultModel:      "gemini-2.5-flash-lite",
		DefaultGenerationConfig: &genaiconfig.GenerationConfig{
			Tools: []*genaiconfig.Tool{
				{
					Name:          "get_time",
					Description:   "Returns the current time for a given city.",
					RequestConfig: &genaiconfig.SchemaConfig{Schema: GetTimeRequest{}},
				},
			},
			ToolConfig: &genaiconfig.ToolConfig{
				// Mode:         genaiconfig.FunctionCallingModeAny,
				// AllowedTools: []string{"get_time"},
			},
		},
	}

	agent, err := genaiClient.NewAgent(ctx, agentConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create agent")
	}
	log.Info().
		Str("agent_id", agentConfig.ID).
		Str("model", agentConfig.DefaultModel).
		Msg("Agent initialized")

	// --- Create Chat ---
	chat, err := agent.NewChat(ctx, &genaiconfig.ChatConfig{
		ID:     "chat_cli_logger_001",
		UserID: USER_ID,
		Model:  agentConfig.DefaultModel,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create chat session")
	}

	fmt.Println("🤖 Chat ready! Type your message (or 'exit' to quit):\n")
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("🧑 You: ")
		userInput, _ := reader.ReadString('\n')
		userInput = strings.TrimSpace(userInput)
		if userInput == "" {
			continue
		}
		if strings.EqualFold(userInput, "exit") {
			fmt.Println("👋 Goodbye!")
			break
		}

		log.Info().Str("prompt", userInput).Msg("User message received")
		start := time.Now()

		// --- Send message synchronously ---
		resp, err := chat.SendMessage(ctx, genaiconfig.Prompt{Text: userInput})
		if err != nil {
			log.Error().Err(err).Msg("Failed to send message")
			continue
		}

		// --- Handle model response ---
		if resp.FunctionCall != nil {
			log.Info().
				Str("function", resp.FunctionCall.Name).
				Interface("args", resp.FunctionCall.Args).
				Msg("Model requested tool call")

			result, err := ExecuteTool(resp.FunctionCall)
			if err != nil {
				log.Error().Err(err).Msg("Tool call failed")
				continue
			}
			fmt.Printf("🧩 Tool result: %s\n", result.Text)
		} else {
			fmt.Printf("🤖 Model: %s\n", resp.Text)
		}

		log.Info().
			Dur("elapsed", time.Since(start)).
			Int("response_len", len(resp.Text)).
			Msg("Message processed")

		// --- Retrieve and log chat history ---
		history, err := chat.GetHistory(ctx)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to get chat history")
			continue
		}
		log.Info().Int("messages", len(history)).Msg("Chat history retrieved")
	}
}
