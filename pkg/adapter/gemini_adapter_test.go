package adapter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/darwishdev/genaiclient/pkg/genaiconfig"
	"google.golang.org/genai"
)

func TestConvertFunctionCallingMode(t *testing.T) {
	tests := []struct {
		name     string
		mode     genaiconfig.FunctionCallingMode
		expected genai.FunctionCallingConfigMode
		wantErr  bool
	}{
		{
			name:     "Unspecified mode",
			mode:     genaiconfig.FunctionCallingModeUnspecified,
			expected: genai.FunctionCallingConfigModeUnspecified,
			wantErr:  false,
		},
		{
			name:     "Auto mode",
			mode:     genaiconfig.FunctionCallingModeAuto,
			expected: genai.FunctionCallingConfigModeAuto,
			wantErr:  false,
		},
		{
			name:     "Any mode",
			mode:     genaiconfig.FunctionCallingModeAny,
			expected: genai.FunctionCallingConfigModeAny,
			wantErr:  false,
		},
		{
			name:     "None mode",
			mode:     genaiconfig.FunctionCallingModeNone,
			expected: genai.FunctionCallingConfigModeNone,
			wantErr:  false,
		},
		{
			name:     "Validated mode",
			mode:     genaiconfig.FunctionCallingModeValidated,
			expected: genai.FunctionCallingConfigModeValidated,
			wantErr:  false,
		},
		{
			name:     "Empty mode defaults to Auto",
			mode:     "",
			expected: genai.FunctionCallingConfigModeAuto,
			wantErr:  false,
		},
		{
			name:     "Invalid mode",
			mode:     "INVALID_MODE",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertFunctionCallingMode(tt.mode)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertFunctionCallingMode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("convertFunctionCallingMode() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBuildGeminiTool(t *testing.T) {
	tests := []struct {
		name    string
		tool    *genaiconfig.Tool
		wantErr bool
	}{
		{
			name: "Basic tool without schemas",
			tool: &genaiconfig.Tool{
				Name:        "testTool",
				Description: "A test tool",
			},
			wantErr: false,
		},
		{
			name: "Tool with JSON schema",
			tool: &genaiconfig.Tool{
				Name:        "weatherTool",
				Description: "Get weather information",
				RequestConfig: &genaiconfig.SchemaConfig{
					SchemaJSON: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"location": map[string]interface{}{
								"type": "string",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Tool with response schema",
			tool: &genaiconfig.Tool{
				Name:        "calculator",
				Description: "Perform calculations",
				ResponseConfig: &genaiconfig.SchemaConfig{
					SchemaJSON: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"result": map[string]interface{}{
								"type": "number",
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := BuildGeminiTool(tt.tool)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildGeminiTool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if result == nil {
					t.Error("BuildGeminiTool() returned nil result")
					return
				}
				if len(result.FunctionDeclarations) != 1 {
					t.Errorf("Expected 1 function declaration, got %d", len(result.FunctionDeclarations))
					return
				}
				if result.FunctionDeclarations[0].Name != tt.tool.Name {
					t.Errorf("Function name = %v, want %v", result.FunctionDeclarations[0].Name, tt.tool.Name)
				}
			}
		})
	}
}

func TestBuildGeminiTools(t *testing.T) {
	tests := []struct {
		name    string
		tools   []*genaiconfig.Tool
		wantErr bool
		wantNil bool
	}{
		{
			name:    "Empty tools list",
			tools:   []*genaiconfig.Tool{},
			wantErr: false,
			wantNil: true,
		},
		{
			name:    "Nil tools list",
			tools:   nil,
			wantErr: false,
			wantNil: true,
		},
		{
			name: "Single tool",
			tools: []*genaiconfig.Tool{
				{
					Name:        "tool1",
					Description: "First tool",
				},
			},
			wantErr: false,
			wantNil: false,
		},
		{
			name: "Multiple tools",
			tools: []*genaiconfig.Tool{
				{
					Name:        "tool1",
					Description: "First tool",
				},
				{
					Name:        "tool2",
					Description: "Second tool",
				},
			},
			wantErr: false,
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := BuildGeminiTools(tt.tools)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildGeminiTools() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantNil && result != nil {
				t.Error("Expected nil result for empty tools")
			}
			if !tt.wantNil && result == nil {
				t.Error("Expected non-nil result")
				return
			}
			if !tt.wantNil && len(result) != len(tt.tools) {
				t.Errorf("Expected %d tools, got %d", len(tt.tools), len(result))
			}
		})
	}
}

func TestGeminiConfigFromGenerationConfig(t *testing.T) {
	temp := float32(0.7)
	topP := float32(0.9)
	topK := float32(40.0)
	maxTokens := int32(1000)

	tests := []struct {
		name    string
		config  *genaiconfig.GenerationConfig
		wantErr bool
	}{
		{
			name:    "Nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "Basic config",
			config: &genaiconfig.GenerationConfig{
				Temperature:     &temp,
				TopP:            &topP,
				TopK:            &topK,
				MaxOutputTokens: maxTokens,
			},
			wantErr: false,
		},
		{
			name: "Config with stop sequences",
			config: &genaiconfig.GenerationConfig{
				StopSequences: []string{"END", "STOP"},
			},
			wantErr: false,
		},
		{
			name: "Config with response schema",
			config: &genaiconfig.GenerationConfig{
				ResponseSchemaConfig: &genaiconfig.SchemaConfig{
					SchemaJSON: map[string]interface{}{
						"type": "object",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Config with tools",
			config: &genaiconfig.GenerationConfig{
				Tools: []*genaiconfig.Tool{
					{
						Name:        "testTool",
						Description: "Test tool",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Config with tool config",
			config: &genaiconfig.GenerationConfig{
				ToolConfig: &genaiconfig.ToolConfig{
					Mode:         genaiconfig.FunctionCallingModeAuto,
					AllowedTools: []string{"tool1", "tool2"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GeminiConfigFromGenerationConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("GeminiConfigFromGenerationConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("Expected non-nil result")
			}
		})
	}
}

func TestGeminiContentFromPrompt(t *testing.T) {
	tests := []struct {
		name    string
		prompt  *genaiconfig.Prompt
		wantErr bool
	}{
		{
			name: "Empty prompt",
			prompt: &genaiconfig.Prompt{
				Text:  "",
				Files: []genaiconfig.FileConfig{},
			},
			wantErr: true,
		},
		{
			name: "Text only prompt",
			prompt: &genaiconfig.Prompt{
				Text: "Hello, world!",
			},
			wantErr: false,
		},
		{
			name: "Structured text prompt",
			prompt: &genaiconfig.Prompt{
				StructuredText: map[string]interface{}{
					"query": "test query",
					"data":  123,
				},
			},
			wantErr: false,
		},
		{
			name: "Combined text and structured text",
			prompt: &genaiconfig.Prompt{
				Text: "Prefix:",
				StructuredText: map[string]interface{}{
					"data": "value",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GeminiContentFromPrompt(tt.prompt)
			if (err != nil) != tt.wantErr {
				t.Errorf("GeminiContentFromPrompt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if result == nil {
					t.Error("Expected non-nil result")
					return
				}
				if len(result) == 0 {
					t.Error("Expected at least one content item")
				}
			}
		})
	}
}

func TestFileConfigToPart(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	testContent := []byte("test content")
	err := os.WriteFile(tmpFile, testContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name    string
		file    genaiconfig.FileConfig
		wantErr bool
	}{
		{
			name: "Inline data",
			file: genaiconfig.FileConfig{
				Contents: []byte("inline data"),
				MIMEType: "text/plain",
			},
			wantErr: false,
		},
		{
			name: "Remote URL",
			file: genaiconfig.FileConfig{
				Path:     "https://example.com/file.pdf",
				MIMEType: "application/pdf",
				Name:     "file.pdf",
			},
			wantErr: false,
		},
		{
			name: "Local file",
			file: genaiconfig.FileConfig{
				Path:     tmpFile,
				MIMEType: "text/plain",
			},
			wantErr: false,
		},
		{
			name: "Empty file config",
			file: genaiconfig.FileConfig{
				Path:     "",
				Contents: nil,
			},
			wantErr: true,
		},
		{
			name: "Non-existent local file",
			file: genaiconfig.FileConfig{
				Path:     "/nonexistent/file.txt",
				MIMEType: "text/plain",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := fileConfigToPart(tt.file)
			if (err != nil) != tt.wantErr {
				t.Errorf("fileConfigToPart() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("Expected non-nil result")
			}
		})
	}
}

func TestIsRemoteURL(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "HTTP URL",
			path: "http://example.com/file",
			want: true,
		},
		{
			name: "HTTPS URL",
			path: "https://example.com/file",
			want: true,
		},
		{
			name: "Local path",
			path: "/local/path/file.txt",
			want: false,
		},
		{
			name: "Relative path",
			path: "./relative/file.txt",
			want: false,
		},
		{
			name: "GCS URL (not detected as remote by this function)",
			path: "gs://bucket/file",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRemoteURL(tt.path); got != tt.want {
				t.Errorf("isRemoteURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModelResponseFromGeminiContent(t *testing.T) {
	tests := []struct {
		name       string
		candidates []*genai.Candidate
		wantErr    bool
		checkFunc  func(*testing.T, *genaiconfig.ModelResponse)
	}{
		{
			name:       "Empty candidates",
			candidates: []*genai.Candidate{},
			wantErr:    true,
		},
		{
			name: "Candidate with no content",
			candidates: []*genai.Candidate{
				{
					Content: nil,
				},
			},
			wantErr: true,
		},
		{
			name: "Candidate with text",
			candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []*genai.Part{
							{Text: "Hello, world!"},
						},
					},
				},
			},
			wantErr: false,
			checkFunc: func(t *testing.T, resp *genaiconfig.ModelResponse) {
				if resp.Text != "Hello, world!" {
					t.Errorf("Expected text 'Hello, world!', got '%s'", resp.Text)
				}
			},
		},
		{
			name: "Candidate with function call",
			candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []*genai.Part{
							{
								FunctionCall: &genai.FunctionCall{
									Name: "testFunction",
									Args: map[string]interface{}{
										"param1": "value1",
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
			checkFunc: func(t *testing.T, resp *genaiconfig.ModelResponse) {
				if resp.FunctionCall == nil {
					t.Error("Expected function call, got nil")
					return
				}
				if resp.FunctionCall.Name != "testFunction" {
					t.Errorf("Expected function name 'testFunction', got '%s'", resp.FunctionCall.Name)
				}
			},
		},
		{
			name: "Multiple parts with text",
			candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []*genai.Part{
							{Text: "Part 1"},
							{Text: "Part 2"},
						},
					},
				},
			},
			wantErr: false,
			checkFunc: func(t *testing.T, resp *genaiconfig.ModelResponse) {
				if resp.Text != "Part 1\nPart 2" {
					t.Errorf("Expected combined text, got '%s'", resp.Text)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ModelResponseFromGeminiContent(tt.candidates)
			if (err != nil) != tt.wantErr {
				t.Errorf("ModelResponseFromGeminiContent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}
}

// // Benchmark tests
func BenchmarkConvertFunctionCallingMode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		convertFunctionCallingMode(genaiconfig.FunctionCallingModeAuto)
	}
}

func BenchmarkBuildGeminiTool(b *testing.B) {
	tool := &genaiconfig.Tool{
		Name:        "testTool",
		Description: "A test tool",
	}
	for i := 0; i < b.N; i++ {
		BuildGeminiTool(tool)
	}
}

func BenchmarkGeminiContentFromPrompt(b *testing.B) {
	prompt := &genaiconfig.Prompt{
		Text: "Hello, world!",
	}
	for i := 0; i < b.N; i++ {
		GeminiContentFromPrompt(prompt)
	}
}
