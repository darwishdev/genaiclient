package adapter

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/darwishdev/genaiclient/pkg/genaiconfig"
	"google.golang.org/genai"
)

// This would use reflection to parse the function's signature and godoc comments.
func FunctionToTool(goFunc interface{}) (*genai.Tool, error) {
	// Implementation detail: Use reflection to build the Tool struct.
	return nil, nil
}

// SchemaForType generates a JSON schema from a Go type.
func SchemaForType(v interface{}) (*genai.Schema, error) {
	// Implementation detail: Use a library like go-jsonschema to generate the schema.
	return nil, nil
}

func ModelResponseFromGeminiContent(res []*genai.Candidate) (genaiconfig.ModelResponse, error) {
	if len(res) == 0 {
		return genaiconfig.ModelResponse{}, errors.New("no candidates found in response")
	}
	c := res[0]
	if c.Content == nil || len(c.Content.Parts) == 0 {
		return genaiconfig.ModelResponse{}, errors.New("candidate has no content parts")
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
	return modelResp, nil
}
