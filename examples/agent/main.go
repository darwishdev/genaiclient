package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/fatih/color"
	"github.com/redis/go-redis/v9"
	"google.golang.org/genai"

	"github.com/darwishdev/genaiclient"
	"github.com/darwishdev/genaiclient/pkg/genaiconfig"
)

// -----------------------------------------------------------
// 🔧 Constants and Prompt Texts
// -----------------------------------------------------------

const (
	USER_ID                = "test-user-123"
	DEFAULT_MODEL          = "gemini-2.5-flash-lite"
	DEFAULT_EMBEDDING_MODE = "gemini-embedding-001"
	USER_PROMPT            = "Explain the difference between a stack and a queue."

	STRUCTURED_PROMPT = `
The latest research indicates a strong correlation between exercise and improved memory retention.
A study published today detailed how 15 minutes of vigorous activity boosted test scores by an average of 10% across all age groups.
This finding suggests a new direction for non-pharmaceutical cognitive enhancement.
`

	TOOL_PROMPT = "What is the temperature like in New York City right now?"
)

// -----------------------------------------------------------
// 🧱 Schema Structs
// -----------------------------------------------------------

type WeatherRequest struct {
	City string `json:"city"`
}

type ArticleSummary struct {
	Title     string   `json:"title"`
	Summary   string   `json:"summary"`
	WordCount int      `json:"word_count"`
	Keywords  []string `json:"keywords"`
}

// -----------------------------------------------------------
// 🏁 Main Entry
// -----------------------------------------------------------

func main() {
	ctx := context.Background()

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		color.Red("FATAL: GEMINI_API_KEY environment variable not set.")
		os.Exit(1)
	}

	// Initialize Gemini + Redis + GenAI
	geminiClient, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey})
	exitIfErr(err, "create Gemini client")

	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 4})

	genaiClient, err := genaiclient.NewGenaiClient(ctx, geminiClient, redisClient, DEFAULT_MODEL, DEFAULT_EMBEDDING_MODE)
	exitIfErr(err, "create GenAI client")

	printHeader("🚀 Starting GenAI Client Demos")

	runSimpleAgent(ctx, genaiClient)
	sectionBreak()
	runToolAgent(ctx, genaiClient)
	sectionBreak()
	runStructuredAgent(ctx, genaiClient)
	sectionBreak()
	runEmbedding(ctx, genaiClient)
	sectionBreak()
	runBulkEmbedding(ctx, genaiClient)
}

// -----------------------------------------------------------
// 🧩 1. Simple Persona Agent
// -----------------------------------------------------------

func runSimpleAgent(ctx context.Context, client genaiclient.GenaiClientInterface) {
	printSection("1️⃣  Simple Agent Demo (Persona Only)")

	agentConfig := genaiconfig.AgentConfig{
		ID:                "Simple-Chatbot",
		Persona:           "You are a helpful assistant named 'Gopher'. Always respond in a friendly and casual tone.",
		SystemInstruction: "Your goal is to be concise and accurate. NEVER write more than 50 words.",
		DefaultModel:      DEFAULT_MODEL,
	}

	agent, err := client.NewAgent(ctx, agentConfig)
	if err != nil {
		color.Red("Error creating agent: %v", err)
		return
	}

	response, err := agent.Generate(ctx, USER_ID, &genaiconfig.Prompt{Text: USER_PROMPT})
	if err != nil {
		color.Red("Error generating content: %v", err)
		return
	}

	color.Green("Agent Response:")
	fmt.Println(response.Text)
}

// -----------------------------------------------------------
// ⚙️ 2. Agent with Function-Calling Tool
// -----------------------------------------------------------

func runToolAgent(ctx context.Context, client genaiclient.GenaiClientInterface) {
	printSection("2️⃣  Tool Agent Demo (Function Calling)")

	tool := &genaiconfig.Tool{
		Name:        "get_current_weather",
		Description: "Retrieves the current weather for a specified city.",
		RequestConfig: &genaiconfig.SchemaConfig{
			Schema: WeatherRequest{},
		},
	}

	agentConfig := genaiconfig.AgentConfig{
		ID:                "Weather-Tool-Agent",
		SystemInstruction: "You are a tool-use expert. Always call 'get_current_weather' for weather-related queries. Respond ONLY with the function call.",
		DefaultModel:      DEFAULT_MODEL,
		DefaultGenerationConfig: &genaiconfig.GenerationConfig{
			Tools: []*genaiconfig.Tool{tool},
			ToolConfig: &genaiconfig.ToolConfig{
				Mode: genaiconfig.FunctionCallingModeAny,
			},
		},
	}

	agent, err := client.NewAgent(ctx, agentConfig)
	if err != nil {
		color.Red("Error creating tool agent: %v", err)
		return
	}

	response, err := agent.Generate(ctx, USER_ID, &genaiconfig.Prompt{Text: TOOL_PROMPT})
	if err != nil {
		color.Red("Error generating content: %v", err)
		return
	}

	if response.FunctionCall != nil {
		color.Green("✅ Tool Call Detected:")
		fmt.Printf("  Function Name: %s\n", response.FunctionCall.Name)
		fmt.Printf("  Arguments: %+v\n", response.FunctionCall.Args)
	} else {
		color.Yellow("⚠️  Expected tool call, got text:")
		fmt.Println(response.Text)
	}
}

// -----------------------------------------------------------
// 🧱 3. Structured JSON Response Agent
// -----------------------------------------------------------

// -----------------------------------------------------------
// 🧱 3. Structured JSON Response Agent
// -----------------------------------------------------------

func runStructuredAgent(ctx context.Context, client genaiclient.GenaiClientInterface) {
	printSection("3️⃣  Structured Response Agent Demo")

	// --- Demonstrate All Three Variations ---

	color.Cyan("→ Variation 1: SchemaJSON (manual map[string]interface{})")
	schemaFromJSON := &genaiconfig.SchemaConfig{
		SchemaJSON: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"city": map[string]interface{}{
					"type": "STRING",
					"enum": []string{"Cairo", "Alexandria", "Luxor", "Aswan"},
				},
				"temperature": map[string]interface{}{
					"type": "NUMBER",
				},
				"condition": map[string]interface{}{
					"type":   "STRING",
					"format": "enum",
					"enum":   []string{"Sunny", "Cloudy", "Rainy"},
				},
			},
			"required": []string{"city", "temperature", "condition"},
		},
	}

	color.Cyan("→ Variation 2: Schema (Go struct reflection)")
	schemaFromStruct := &genaiconfig.SchemaConfig{
		Schema: WeatherRequest{},
	}

	color.Cyan("→ Variation 3: SchemaGenAI (direct *genai.Schema)")
	schemaFromGenAI := &genaiconfig.SchemaConfig{
		SchemaGenAI: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"city":        {Type: genai.TypeString, Enum: []string{"Cairo", "Alexandria", "Luxor"}},
				"temperature": {Type: genai.TypeNumber},
				"condition":   {Type: genai.TypeString, Enum: []string{"Sunny", "Cloudy", "Rainy"}},
			},
			Required: []string{"city", "temperature", "condition"},
		},
	}

	var configs = []*genaiconfig.SchemaConfig{schemaFromJSON, schemaFromStruct, schemaFromGenAI}

	for i, cfg := range configs {
		color.Magenta("\n🔹 Running Structured Demo Variation #%d", i+1)

		agentConfig := genaiconfig.AgentConfig{
			ID:                fmt.Sprintf("Structured-Agent-%d", i+1),
			SystemInstruction: "You are a structured data generator that outputs weather information following the provided schema exactly.",
			DefaultModel:      DEFAULT_MODEL,
			DefaultGenerationConfig: &genaiconfig.GenerationConfig{
				ResponseSchemaConfig: cfg,
			},
		}

		agent, err := client.NewAgent(ctx, agentConfig)
		if err != nil {
			color.Red("Error creating structured agent: %v", err)
			continue
		}

		response, err := agent.Generate(ctx, USER_ID, &genaiconfig.Prompt{
			Text: "Generate fake weather data for Cairo.",
		})
		if err != nil {
			color.Red("Error generating structured response: %v", err)
			continue
		}

		color.Yellow("Raw Response (should be valid JSON):")
		fmt.Println(response.Text)

		// Try to unmarshal dynamically
		var generic map[string]interface{}
		if err := json.Unmarshal([]byte(response.Text), &generic); err != nil {
			color.Red("❌ JSON parse failed: %v", err)
			continue
		}

		color.Green("✅ Parsed structured data successfully:")
		for k, v := range generic {
			fmt.Printf("  • %s: %v\n", k, v)
		}
	}
}

// -----------------------------------------------------------
// 🧬 4. Single Embedding
// -----------------------------------------------------------

func runEmbedding(ctx context.Context, client genaiclient.GenaiClientInterface) {
	printSection("4️⃣  Single Embedding Demo")

	text := "This is a sentence about cats and dogs."
	vecs, err := client.Embed(ctx, text)
	if err != nil {
		color.Red("Error generating embedding: %v", err)
		return
	}

	if len(vecs) > 0 && len(vecs[0]) > 0 {
		color.Green("✅ Embedding Generated")
		fmt.Printf("  Text: %q\n  Vector Length: %d\n  Preview: %v...\n", text, len(vecs[0]), vecs[0][:5])
	}
}

// -----------------------------------------------------------
// 🧩 5. Bulk Embedding
// -----------------------------------------------------------

func runBulkEmbedding(ctx context.Context, client genaiclient.GenaiClientInterface) {
	printSection("5️⃣  Bulk Embedding Demo")

	texts := []string{
		"The quick brown fox jumps over the lazy dog.",
		"A new era of machine learning is upon us.",
		"Go is a statically typed, compiled programming language.",
	}

	vecs, err := client.EmbedBulk(ctx, texts)
	if err != nil {
		color.Red("Error generating bulk embeddings: %v", err)
		return
	}

	color.Green("✅ Embedded %d texts. Example vector size: %d", len(vecs), len(vecs[0][0]))
	fmt.Printf("  First vector start: %v...\n", vecs[0][0][:3])
}

// -----------------------------------------------------------
// 🧰 Helpers
// -----------------------------------------------------------

func buildGenAISchemaFromType(t reflect.Type) *genai.Schema {
	// Simplified for demo — in your code, this already exists.
	return &genai.Schema{Type: genai.TypeObject}
}

func exitIfErr(err error, msg string) {
	if err != nil {
		color.Red("Failed to %s: %v", msg, err)
		os.Exit(1)
	}
}

func sectionBreak() {
	fmt.Println(strings.Repeat("-", 60))
}

func printHeader(s string) {
	color.HiBlue("\n%s\n%s", s, strings.Repeat("=", len(s)+2))
}

func printSection(s string) {
	color.HiCyan("\n%s\n%s", s, strings.Repeat("-", len(s)+2))
}

func runCLIChat(ctx context.Context, client genaiclient.GenaiClientInterface) {
	printSection("🖥️ CLI Chat (Streaming + Tools)")

	// Define tools
	tools := []*genaiconfig.Tool{
		{
			Name:        "get_current_weather",
			Description: "Gets the current temperature of a given city.",
			RequestConfig: &genaiconfig.SchemaConfig{
				Schema: WeatherRequest{},
			},
		},
		{
			Name:        "summarize_text",
			Description: "Summarizes a given paragraph into 2-3 sentences.",
			RequestConfig: &genaiconfig.SchemaConfig{
				Schema: ArticleSummary{},
			},
		},
	}

	agentCfg := genaiconfig.AgentConfig{
		ID:                "CLI-Agent",
		Persona:           "You are a helpful assistant that uses tools when needed.",
		SystemInstruction: "Always respond concisely, and call tools where relevant.",
		DefaultModel:      DEFAULT_MODEL,
		DefaultGenerationConfig: &genaiconfig.GenerationConfig{
			Tools: tools,
			ToolConfig: &genaiconfig.ToolConfig{
				Mode: genaiconfig.FunctionCallingModeAuto,
			},
		},
	}

	agent, err := client.NewAgent(ctx, agentCfg)
	exitIfErr(err, "create CLI agent")

	chat, err := agent.NewChat(ctx, USER_ID)
	exitIfErr(err, "create CLI chat")

	color.Green("Available Tools:")
	for _, t := range tools {
		fmt.Printf("  • %s: %s\n", t.Name, t.Description)
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(color.HiYellowString("\nYou > "))
		msg, _ := reader.ReadString('\n')
		msg = strings.TrimSpace(msg)
		if msg == "exit" || msg == "quit" {
			color.Cyan("👋 Goodbye!")
			return
		}
		if msg == "" {
			continue
		}

		color.Blue("Model > (streaming...)")
		stream, err := chat.Stream(ctx, msg)
		if err != nil {
			color.Red("Stream error: %v", err)
			continue
		}

		for chunk := range stream {
			if chunk.Error != nil {
				color.Red("Stream chunk error: %v", chunk.Error)
				break
			}
			if chunk.Text != "" {
				fmt.Print(chunk.Text)
			}
			if chunk.FunctionCall != nil {
				color.Magenta("\n🧰 Tool Call: %s %+v\n", chunk.FunctionCall.Name, chunk.FunctionCall.Args)
			}
		}
		fmt.Println()
	}
}
