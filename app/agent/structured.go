package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/darwishdev/genaiclient"
	"github.com/darwishdev/genaiclient/pkg/genaiconfig"
)

type StructuredAgentInterface[T any] interface {
	GenerateContent(ctx context.Context, prompt string) (T, error)
	UpdateConfig(ctx context.Context, persona string, systemInstruction string) error
	ID() string
}
type structuredAgent[T any] struct {
	id         string
	agent      AgentInterface
	schemaType reflect.Type
	client     genaiclient.GenaiClientInterface
}

func NewStructuredAgent[T any](
	ctx context.Context,
	client genaiclient.GenaiClientInterface,
	id string,
	persona string,
	systemInstruction string,
	model string,
) (StructuredAgentInterface[T], error) {

	var schemaInstance T
	temp := float32(.01)
	agentConfig := genaiconfig.AgentConfig{
		ID:                id,
		Persona:           persona,
		SystemInstruction: systemInstruction,
		DefaultModel:      model,
		DefaultGenerationConfig: &genaiconfig.GenerationConfig{
			Temperature: &temp,
			ResponseSchemaConfig: &genaiconfig.SchemaConfig{
				Schema: schemaInstance,
			},
		},
	}

	ag, err := client.NewAgent(ctx, agentConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create structured agent: %w", err)
	}

	return &structuredAgent[T]{
		id:         id,
		agent:      ag,
		schemaType: reflect.TypeOf(schemaInstance),
		client:     client,
	}, nil
}

func (s *structuredAgent[T]) GenerateContent(ctx context.Context, prompt string) (T, error) {
	var zero T
	response, err := s.agent.Generate(ctx, "user-id-placeholder", &genaiconfig.Prompt{Text: prompt})
	if err != nil {
		return zero, err
	}

	output := reflect.New(s.schemaType).Interface()
	if err := json.Unmarshal([]byte(response.Text), output); err != nil {
		return zero, fmt.Errorf("failed to parse structured output: %w", err)
	}

	return *(output.(*T)), nil
}

func (s *structuredAgent[T]) UpdateConfig(ctx context.Context, persona string, systemInstruction string) error {
	var schemaInstance T
	agentConfig := genaiconfig.AgentConfig{
		Persona:           persona,
		SystemInstruction: systemInstruction,
		DefaultGenerationConfig: &genaiconfig.GenerationConfig{
			ResponseSchemaConfig: &genaiconfig.SchemaConfig{
				Schema: schemaInstance,
			},
		},
	}
	_, err := s.client.NewAgent(ctx, agentConfig)
	if err != nil {
		return fmt.Errorf("failed to create structured agent: %w", err)
	}
	return nil
}

func (s *structuredAgent[T]) ID() string {
	return s.id
}
