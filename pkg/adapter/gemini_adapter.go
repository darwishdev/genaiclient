package adapter

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/darwishdev/genaiclient/pkg/genaiconfig"
	"google.golang.org/genai"
)

func convertFunctionCallingMode(mode genaiconfig.FunctionCallingMode) (genai.FunctionCallingConfigMode, error) {
	switch mode {
	case genaiconfig.FunctionCallingModeUnspecified:
		return genai.FunctionCallingConfigModeUnspecified, nil
	case genaiconfig.FunctionCallingModeAuto:
		return genai.FunctionCallingConfigModeAuto, nil
	case genaiconfig.FunctionCallingModeAny:
		return genai.FunctionCallingConfigModeAny, nil
	case genaiconfig.FunctionCallingModeNone:
		return genai.FunctionCallingConfigModeNone, nil
	case genaiconfig.FunctionCallingModeValidated:
		return genai.FunctionCallingConfigModeValidated, nil
	case "": // Empty mode defaults to Auto
		return genai.FunctionCallingConfigModeAuto, nil
	default:
		return "", fmt.Errorf("invalid function calling mode: %s", mode)
	}
}
func BuildGeminiTool(tool *genaiconfig.Tool) (*genai.Tool, error) {
	functionDeclaration := genai.FunctionDeclaration{
		Name:        tool.Name,
		Description: tool.Description,
	}
	requestConfig := tool.RequestConfig
	responseConfig := tool.ResponseConfig
	if requestConfig != nil {
		if requestConfig.SchemaJSON != nil {
			functionDeclaration.ParametersJsonSchema = requestConfig.SchemaJSON
		}
		if requestConfig.Schema != nil {
			functionDeclaration.Parameters = buildSchemaFromType(reflect.TypeOf(requestConfig.Schema))
		}
		if requestConfig.SchemaGenAI != nil {
			functionDeclaration.Parameters = requestConfig.SchemaGenAI
		}
	}
	if responseConfig != nil {
		if responseConfig.SchemaJSON != nil {
			functionDeclaration.ResponseJsonSchema = responseConfig.SchemaJSON
		}
		if responseConfig.Schema != nil {
			functionDeclaration.Parameters = buildSchemaFromType(reflect.TypeOf(responseConfig.Schema))
		}
		if responseConfig.SchemaGenAI != nil {
			functionDeclaration.Parameters = responseConfig.SchemaGenAI
		}
	}
	return &genai.Tool{
		FunctionDeclarations: []*genai.FunctionDeclaration{&functionDeclaration},
	}, nil

}
func BuildGeminiTools(tools []*genaiconfig.Tool) ([]*genai.Tool, error) {
	if len(tools) == 0 {
		return nil, nil
	}
	geminiTools := make([]*genai.Tool, len(tools))
	for index, tool := range tools {
		geminiTool, err := BuildGeminiTool(tool)
		if err != nil {
			return nil, fmt.Errorf("Eror Converting tool %s to gemini tool: %w", tool.Name, err)
		}
		geminiTools[index] = geminiTool
	}
	return geminiTools, nil
}
func GeminiConfigFromGenerationConfig(config *genaiconfig.GenerationConfig) (*genai.GenerateContentConfig, error) {
	if config == nil {
		return nil, fmt.Errorf("Config is null please provide config")
	}
	genConfig := &genai.GenerateContentConfig{
		Temperature:     config.Temperature,
		TopP:            config.TopP,
		TopK:            config.TopK,
		MaxOutputTokens: config.MaxOutputTokens,
		StopSequences:   config.StopSequences,
	}

	// Convert ResponseJSONSchema map to genai.Schema
	responseSchema := config.ResponseSchemaConfig
	if responseSchema != nil {
		genConfig.ResponseMIMEType = "application/json"
		if responseSchema.SchemaJSON != nil {
			genConfig.ResponseJsonSchema = responseSchema.SchemaJSON
		}
		if responseSchema.Schema != nil {
			genConfig.ResponseSchema = buildSchemaFromType(reflect.TypeOf(responseSchema.Schema))
		}
		if responseSchema.SchemaGenAI != nil {
			genConfig.ResponseSchema = responseSchema.SchemaGenAI
		}
	}

	// Convert Tools from simple format to Gemini's FunctionDeclaration format
	if len(config.Tools) > 0 {
		tools, err := BuildGeminiTools(config.Tools)
		if err != nil {
			return nil, fmt.Errorf("Unable to convert tools to gemini tools :%w", err)
		}
		genConfig.Tools = tools
	}

	// ToolConfig can be assigned directly if types match
	if config.ToolConfig != nil {
		mode, err := convertFunctionCallingMode(config.ToolConfig.Mode)
		if err != nil {
			return nil, fmt.Errorf("Unable to convert tool config calling mode to gemini:%w", err)
		}
		genConfig.ToolConfig = &genai.ToolConfig{
			FunctionCallingConfig: &genai.FunctionCallingConfig{
				Mode:                 mode,
				AllowedFunctionNames: config.ToolConfig.AllowedTools,
			}}
	}

	return genConfig, nil
}
func GeminiContentFromPrompt(prompt *genaiconfig.Prompt) ([]*genai.Content, error) {
	if prompt.Text == "" && len(prompt.Files) == 0 && prompt.StructuredText == nil {
		return nil, errors.New("prompt must contain at least text, structured text, or files")
	}

	parts := make([]*genai.Part, 0)

	// 1. Handle text content
	if prompt.Text != "" {
		parts = append(parts, &genai.Part{
			Text: prompt.Text,
		})
	}

	// 2. Handle structured text (JSON)
	if prompt.StructuredText != nil {
		jsonBytes, err := json.Marshal(prompt.StructuredText)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal structured text: %w", err)
		}
		parts = append(parts, &genai.Part{
			Text: string(jsonBytes),
		})
	}

	// 3. Handle files
	for i, file := range prompt.Files {
		part, err := fileConfigToPart(file)
		if err != nil {
			return nil, fmt.Errorf("failed to process file at index %d: %w", i, err)
		}
		parts = append(parts, part)
	}

	content := &genai.Content{
		Parts: parts,
		Role:  "user", // Default role for user prompts
	}

	return []*genai.Content{content}, nil
}

func fileConfigToPart(file genaiconfig.FileConfig) (*genai.Part, error) {
	part := &genai.Part{}

	// Determine MIME type (fallback to default)
	mimeType := file.MIMEType
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Case 1: Explicit inline data provided
	if len(file.Contents) > 0 {
		part.InlineData = &genai.Blob{
			MIMEType: mimeType,
			Data:     file.Contents,
		}
		return part, nil
	}

	// Case 2: Handle file path
	if file.Path != "" {
		isRemote := isRemoteURL(file.Path)

		if isRemote {
			// Remote URI (e.g. gs://, https://, s3://)
			part.FileData = &genai.FileData{
				DisplayName: file.Name,
				FileURI:     file.Path,
				MIMEType:    mimeType,
			}
			return part, nil
		}

		// Local file → read contents as inline data
		data, err := os.ReadFile(filepath.Clean(file.Path))
		if err != nil {
			return nil, fmt.Errorf("fileConfigToPart: failed to read local file %q: %w", file.Path, err)
		}

		part.InlineData = &genai.Blob{
			MIMEType: mimeType,
			Data:     data,
		}
		return part, nil
	}

	return nil, fmt.Errorf("fileConfigToPart: both file.Contents and file.Path are empty")
}

// isRemoteURL checks if a path is a remote URL
func isRemoteURL(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}

func ModelResponseFromGeminiContent(res []*genai.Candidate) (*genaiconfig.ModelResponse, error) {
	if len(res) == 0 {
		return nil, errors.New("no candidates found in response")
	}
	c := res[0]
	if c.Content == nil || len(c.Content.Parts) == 0 {
		return nil, errors.New("candidate has no content parts")
	}

	var sb strings.Builder
	var modelResp genaiconfig.ModelResponse
	for _, part := range c.Content.Parts {
		if part == nil {
			continue
		}

		switch {
		case part.FunctionCall != nil:
			modelResp.FunctionCall = &genaiconfig.FunctionCall{
				Name: part.FunctionCall.Name,
				Args: part.FunctionCall.Args,
			}

		case part.Text != "":
			sb.WriteString(part.Text)
			sb.WriteString("\n")

		default:
			raw, err := json.Marshal(part)
			if err == nil {
				sb.WriteString(string(raw))
				sb.WriteString("\n")
			}
		}
	}
	modelResp.Text = strings.TrimSpace(sb.String())
	return &modelResp, nil
}
