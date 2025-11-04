package test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"google.golang.org/genai"

	"github.com/darwishdev/genaiclient"
	"github.com/darwishdev/genaiclient/pkg/genaiconfig"
)

// -----------------------------------------------------------
// 🧱 Schema Struct
// -----------------------------------------------------------

type WeatherRequest struct {
	City string `json:"city"`
}

// -----------------------------------------------------------
// ⚙️ Setup Helpers
// -----------------------------------------------------------

const (
	DEFAULT_MODEL          = "gemini-2.5-flash-lite"
	DEFAULT_EMBEDDING_MODE = "gemini-embedding-001"
	USER_ID                = "integration-test-user"
)

func newRealClient(t *testing.T) genaiclient.GenaiClientInterface {
	ctx := context.Background()

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping integration test: GEMINI_API_KEY not set")
	}

	// ✅ match your main() style
	geminiClient, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey})
	if err != nil {
		t.Fatalf("Failed to create Gemini client: %v", err)
	}

	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 5})

	client, err := genaiclient.NewGenaiClient(ctx, geminiClient, redisClient, DEFAULT_MODEL, DEFAULT_EMBEDDING_MODE)
	if err != nil {
		t.Fatalf("Failed to create GenAI client: %v", err)
	}
	return client
}

// -----------------------------------------------------------
// 🧱 Example Structs
// -----------------------------------------------------------

type AddRequest struct {
	A int `json:"a"`
	B int `json:"b"`
}

type AddResponse struct {
	Sum int `json:"sum"`
}

func TestIntegration_Generate_WithSchemas(t *testing.T) {
	client := newRealClient(t)
	ctx := context.Background()

	type TestCase struct {
		Name        string
		Prompt      string
		Schema      *genai.Schema
		ExpectKeys  []string
		ExpectTypes map[string]string
	}

	tests := []TestCase{
		{
			Name:   "Square Calculator",
			Prompt: "Find the square of 7 using the squarer tool.",
			Schema: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"square": {Type: genai.TypeNumber},
				},
				Required: []string{"square"},
			},
			ExpectKeys:  []string{"square"},
			ExpectTypes: map[string]string{"square": "float64"},
		},
		{
			Name:   "Weather Info",
			Prompt: "Give me the weather for Cairo city in a structured JSON.",
			Schema: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"city":        {Type: genai.TypeString},
					"temperature": {Type: genai.TypeNumber},
					"condition":   {Type: genai.TypeString},
				},
				Required: []string{"city", "temperature"},
			},
			ExpectKeys:  []string{"city", "temperature", "condition"},
			ExpectTypes: map[string]string{"city": "string", "temperature": "float64", "condition": "string"},
		},
		{
			Name:   "Simple Math Result",
			Prompt: "Add 3 and 4 and return JSON with their sum.",
			Schema: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"sum": {Type: genai.TypeNumber},
				},
				Required: []string{"sum"},
			},
			ExpectKeys:  []string{"sum"},
			ExpectTypes: map[string]string{"sum": "float64"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			agentCfg := genaiconfig.AgentConfig{
				ID:                "Agent-" + tt.Name,
				SystemInstruction: "Return only JSON according to the schema.",
				DefaultModel:      DEFAULT_MODEL,
				DefaultGenerationConfig: &genaiconfig.GenerationConfig{
					ResponseSchemaConfig: &genaiconfig.SchemaConfig{
						SchemaGenAI: tt.Schema,
					},
				},
			}

			agent, err := client.NewAgent(ctx, agentCfg)
			assert.NoError(t, err)
			assert.NotNil(t, agent)

			resp, err := agent.Generate(ctx, USER_ID, &genaiconfig.Prompt{
				Text: tt.Prompt,
			})
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			assert.NotEmpty(t, resp.Text)

			// 🔍 Validate JSON format
			var parsed map[string]any
			err = json.Unmarshal([]byte(resp.Text), &parsed)
			if err != nil {
				t.Fatalf("Response is not valid JSON: %v\nResponse Text: %s", err, resp.Text)
			}

			// ✅ Ensure all expected keys exist
			for _, key := range tt.ExpectKeys {
				_, ok := parsed[key]
				assert.True(t, ok, "Expected key %q not found in response: %v", key, parsed)
			}

			// ✅ Ensure expected types match
			for key, wantType := range tt.ExpectTypes {
				if val, ok := parsed[key]; ok {
					gotType := fmt.Sprintf("%T", val)
					assert.Equal(t, wantType, gotType, "Unexpected type for key %q", key)
				}
			}

			// 🪶 Optional debug log
			t.Logf("[%s] Raw JSON: %s", tt.Name, resp.Text)
		})
	}
}
