package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/redis/go-redis/v9"
	"google.golang.org/genai"

	"github.com/darwishdev/genaiclient"
	"github.com/darwishdev/genaiclient/pkg/genaiconfig"
)

func main() {
	ctx := context.Background()

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		panic("GEMINI_API_KEY not set")
	}

	// --- Setup Clients ---
	geminiClient, _ := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey})
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 10})
	client, _ := genaiclient.NewGenaiClient(ctx, geminiClient, redisClient, "gemini-2.5-flash-lite", "gemini-embedding-001")

	// --- Create Agent ---
	// agent, _ := client.NewAgent(ctx, genaiconfig.AgentConfig{
	// 	ID:           "SimpleChatAgent",
	// 	Persona:      "You are a concise assistant named GoBot.",
	// 	DefaultModel: "gemini-2.5-flash-lite",
	// })

	// --- Create Chat Session ---
	// chat, _ := agent.NewChat(ctx, &genaiconfig.ChatConfig{
	// 	ID:     "simple-chat-session",
	// 	UserID: "user-1",
	// 	Model:  "gemini-2.5-flash-lite",
	// })

	// // --- Send Message ---
	// resp, _ := chat.SendMessage(ctx, genaiconfig.Prompt{Text: "Hello! Who are you?"})
	// color.Green("Response: %s", resp.Text)
	//
	// toolChatDemo(ctx, client)
	// structuredChatDemo(ctx, client)
	cliChatDemo(ctx, client)
	// cliChatNormalDemo(ctx, client)
}

func toolChatDemo(ctx context.Context, client genaiclient.GenaiClientInterface) {
	color.HiCyan("\n2️⃣ Conversational Tool Chat Demo")

	// --- Define the weather tool ---
	weatherTool := &genaiconfig.Tool{
		Name:        "get_weather",
		Description: "Fetch the current temperature and condition for a given city.",
		RequestConfig: &genaiconfig.SchemaConfig{
			Schema: struct {
				City string `json:"city"`
			}{},
		},
	}

	// --- Create the agent ---
	agent, _ := client.NewAgent(ctx, genaiconfig.AgentConfig{
		ID:           "ConversationalToolAgent",
		DefaultModel: "gemini-2.5-flash-lite",
		SystemInstruction: `
You are GoBot, a friendly weather assistant.
You can talk casually with users but should call the "get_weather" tool
whenever you need to fetch weather data for a city.
After receiving tool results, summarize them in a conversational tone.
		`,
		DefaultGenerationConfig: &genaiconfig.GenerationConfig{
			Tools: []*genaiconfig.Tool{weatherTool},
		},
	})

	// --- Create chat session ---
	chat, _ := agent.NewChat(ctx, &genaiconfig.ChatConfig{
		ID:     "tool-chat",
		UserID: "user-1",
		Model:  "gemini-2.5-flash-lite",
	})

	color.Yellow("👋 GoBot is ready! Ask me about any city's weather.\n")

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(color.HiWhiteString("\nYou: "))
		input, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		input = strings.TrimSpace(input)
		if input == "exit" {
			color.Red("Goodbye!")
			break
		}

		// --- Send message to the model ---
		resp, err := chat.SendMessage(ctx, genaiconfig.Prompt{Text: input})
		if err != nil {
			color.Red("Error: %v", err)
			continue
		}

		// --- If model called the tool ---
		if resp.FunctionCall != nil {
			color.HiMagenta("\n🧰 GoBot called: %s", resp.FunctionCall.Name)
			fmt.Printf("Args: %+v\n", resp.FunctionCall.Args)

			// Simulate weather API result
			city := resp.FunctionCall.Args["city"].(string)
			mockResult := map[string]interface{}{
				"city":        city,
				"temperature": "18°C",
				"condition":   "Partly cloudy",
			}

			color.Yellow("↩️ Sending tool result back to GoBot...")

			finalResp, err := chat.SendToolResponse(ctx, *resp.FunctionCall, mockResult)
			if err != nil {
				color.Red("Error sending tool response: %v", err)
				continue
			}
			color.Green("💬 GoBot: %s", finalResp.Text)

		} else {
			// No tool used, just chat response
			color.Green("💬 GoBot: %s", resp.Text)
		}
	}
}

func structuredChatDemo(ctx context.Context, client genaiclient.GenaiClientInterface) {
	color.HiCyan("\n3️⃣ Structured Chat Demo")

	agent, _ := client.NewAgent(ctx, genaiconfig.AgentConfig{
		ID:           "StructuredAgent",
		DefaultModel: "gemini-2.5-flash-lite",
	})

	overrideConfig := &genaiconfig.GenerationConfig{
		ResponseSchemaConfig: &genaiconfig.SchemaConfig{
			Schema: struct {
				City        string  `json:"city"`
				Temperature float64 `json:"temperature"`
				Condition   string  `json:"condition"`
			}{},
		},
	}
	chat, _ := agent.NewChat(ctx, &genaiconfig.ChatConfig{
		ID:               "structured-chat",
		UserID:           "user-1",
		GenerationConfig: overrideConfig,
		Model:            "gemini-2.5-flash-lite",
	})

	resp, _ := chat.SendMessage(ctx, genaiconfig.Prompt{
		Text: "Generate fake weather data for Cairo.",
	})

	color.Green("Structured JSON:\n%s", resp.Text)
}

func cliChatNormalDemo(ctx context.Context, client genaiclient.GenaiClientInterface) {
	color.HiCyan("\n5️⃣ CLI Normal Chat Demo (non-streaming)")

	// Define tools
	tools := []*genaiconfig.Tool{
		{
			Name:        "search_news",
			Description: "Search latest news headlines.",
			RequestConfig: &genaiconfig.SchemaConfig{
				Schema: struct {
					Query string `json:"query"`
				}{},
			},
		},
		{
			Name:        "get_time",
			Description: "Get the current time for a given city.",
			RequestConfig: &genaiconfig.SchemaConfig{
				Schema: struct {
					City string `json:"city"`
				}{},
			},
		},
	}

	// Print available tools
	color.Yellow("Available Tools:")
	for _, t := range tools {
		fmt.Printf(" - %s: %s\n", t.Name, t.Description)
	}

	// Create agent
	agent, _ := client.NewAgent(ctx, genaiconfig.AgentConfig{
		ID:                "CLI-Agent-Normal",
		DefaultModel:      "gemini-2.5-flash-lite",
		SystemInstruction: "You are a CLI assistant. Use available tools when appropriate.",
		DefaultGenerationConfig: &genaiconfig.GenerationConfig{
			Tools: tools,
			// ToolConfig: &genaiconfig.ToolConfig{
			// 	Mode: genaiconfig.FunctionCallingModeAny,
			// },
		},
	})

	// Create chat
	chat, _ := agent.NewChat(ctx, &genaiconfig.ChatConfig{
		ID:     "cli-chat-normal",
		UserID: "user-cli",
		Model:  "gemini-2.5-flash-lite",
	})

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(color.HiWhiteString("\nYou: "))
		input, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		input = strings.TrimSpace(input)
		if input == "exit" {
			color.Red("Exiting chat...")
			break
		}

		resp, err := chat.SendMessage(ctx, genaiconfig.Prompt{Text: input})
		if err != nil {
			color.Red("Error: %v", err)
			continue
		}

		// Handle tool call
		if resp.FunctionCall != nil {
			color.HiMagenta("\n🧰 Tool Call: %s", resp.FunctionCall.Name)
			fmt.Printf("Args: %+v\n", resp.FunctionCall.Args)

			result := ""
			switch resp.FunctionCall.Name {
			case "search_news":
				query, _ := resp.FunctionCall.Args["query"].(string)
				result = fmt.Sprintf("Found 3 news headlines for '%s'.", query)
			case "get_time":
				city, _ := resp.FunctionCall.Args["city"].(string)
				result = fmt.Sprintf("The time in %s is 3:45 PM.", city)
			default:
				result = "Unknown tool."
			}

			color.Yellow("↩️ Sending tool result back to model...")
			finalResp, err := chat.SendToolResponse(ctx, *resp.FunctionCall, result)
			if err != nil {
				color.Red("Error sending tool response: %v", err)
				continue
			}
			color.Green("🧠 Model (after tool): %s\n", finalResp.Text)
		} else {
			color.Green("💬 Model: %s", resp.Text)
		}
	}
}
func cliChatDemo(ctx context.Context, client genaiclient.GenaiClientInterface) {
	color.HiCyan("\n4️⃣ CLI Streaming Chat Demo")

	// Define a couple of tools
	tools := []*genaiconfig.Tool{
		{
			Name:        "search_news",
			Description: "Search latest news headlines.",
			RequestConfig: &genaiconfig.SchemaConfig{
				Schema: struct {
					Query string `json:"query"`
				}{},
			},
		},
		{
			Name:        "get_time",
			Description: "Get the current time for a given city.",
			RequestConfig: &genaiconfig.SchemaConfig{
				Schema: struct {
					City string `json:"city"`
				}{},
			},
		},
	}

	// Print available tools
	color.Yellow("Available Tools:")
	for _, t := range tools {
		fmt.Printf(" - %s: %s\n", t.Name, t.Description)
	}

	// Create agent
	agent, _ := client.NewAgent(ctx, genaiconfig.AgentConfig{
		ID:                "CLI-Agent",
		DefaultModel:      "gemini-2.5-flash-lite",
		SystemInstruction: "You are a CLI chat assistant. Use available tools where appropriate.",
		DefaultGenerationConfig: &genaiconfig.GenerationConfig{
			Tools: tools,
			// ToolConfig: &genaiconfig.ToolConfig{
			// 	Mode: genaiconfig.FunctionCallingModeAny,
			// },
		},
	})

	// Create chat
	chat, _ := agent.NewChat(ctx, &genaiconfig.ChatConfig{
		ID:     "cli-chat-session",
		UserID: "user-cli",
		Model:  "gemini-2.5-flash-lite",
	})

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(color.HiWhiteString("\nYou: "))

		input, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		input = strings.TrimSpace(input)
		if input == "exit" {
			color.Red("Exiting chat...")
			break
		}

		stream, _ := chat.SendMessageStream(ctx, genaiconfig.Prompt{Text: input})
		fmt.Print(color.GreenString("Model: "))

		var finalResp *genaiconfig.ModelResponse
		for chunk := range stream {
			if chunk.Error != nil {
				color.Red("Error: %v", chunk.Error)
				break
			}

			if chunk.FunctionCall != nil {
				color.HiMagenta("\n🧰 Tool Call: %s", chunk.FunctionCall.Name)
				fmt.Printf("Args: %+v\n", chunk.FunctionCall.Args)

				// --- Mock execution ---
				result := ""
				switch chunk.FunctionCall.Name {
				case "search_news":
					query, _ := chunk.FunctionCall.Args["query"].(string)
					result = fmt.Sprintf("Found 3 news headlines for '%s'.", query)
				case "get_time":
					city, _ := chunk.FunctionCall.Args["city"].(string)
					result = fmt.Sprintf("The time in %s is 3:45 PM.", city)
				default:
					result = "Unknown tool."
				}

				color.Yellow("↩️ Sending tool result back to model...")
				finalResp, err = chat.SendToolResponse(ctx, *chunk.FunctionCall, result)
				if err != nil {
					color.Red("Error sending tool response: %v", err)
					break
				}

				color.Green("🧠 Model (after tool): %s\n", finalResp.Text)
			} else {
				fmt.Print(chunk.Text)
				finalResp = chunk
			}
		}
		fmt.Println()

		if finalResp != nil && finalResp.FunctionCall == nil {
			color.Green("💬 Final Response: %s", finalResp.Text)
		}
	}

}
