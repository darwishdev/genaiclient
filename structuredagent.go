package genaiclient

import (
	"context"
	"fmt"
	"reflect"

	"github.com/darwishdev/genaiclient/pkg/adapter"
	"github.com/redis/go-redis/v9"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/genai"
)

type GenAIStructuredAgentInterface[TReq any, TRes any] interface {
	NewInMemorySession(ctx context.Context, userID string) GenAIStructuredSessionInterface[TReq, TRes]
	NewVertexSession(
		ctx context.Context,
		userID string,
	) (GenAIStructuredSessionInterface[TReq, TRes], error)
	NewRedisSession(
		ctx context.Context,
		userID string,
		sessionID string,
		rdb *redis.Client,
	) (GenAIStructuredSessionInterface[TReq, TRes], error)
}

type GenAIStructuredAgent[TReq any, TRes any] struct {
	base      GenAIAgentInterface
	outputKey string
}

func isEmptyStruct[T any]() bool {
	var t T
	tp := reflect.TypeOf(t)
	return tp.Kind() == reflect.Struct && tp.NumField() == 0
}
func NewStructuredAgent[TReq any, TRes any](
	appName string,
	apiKey string,
	modelName string,
	agentName string,
	agentDescription string,
	agentInstructions string,
	enableTracer bool,
	overrideCfg ...llmagent.Config,
) (GenAIStructuredAgentInterface[TReq, TRes], error) {
	outputKey := "result"
	var cfg llmagent.Config
	if len(overrideCfg) > 0 {
		cfg = overrideCfg[0] // user-supplied config
	}
	cfg.Name = agentName
	cfg.Description = agentDescription
	cfg.Instruction = agentInstructions
	if !isEmptyStruct[TRes]() {
		var tr TRes
		cfg.OutputSchema = adapter.BuildSchemaFromStruct(tr)
	}
	if !isEmptyStruct[TReq]() {
		var tq TReq
		cfg.InputSchema = adapter.BuildSchemaFromStruct(tq)
	}
	if enableTracer {
		before, after := EnableTracer()
		cfg.BeforeModelCallbacks = append(cfg.BeforeModelCallbacks, before)
		cfg.AfterModelCallbacks = append(cfg.AfterModelCallbacks, after)
	}
	ctx := context.Background()
	m, err := gemini.NewModel(ctx, modelName, &genai.ClientConfig{APIKey: apiKey})
	if err != nil {
		return nil, fmt.Errorf("model error: %w", err)
	}
	cfg.Model = m
	baseAgent, err := llmagent.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("agent build error: %w", err)
	}
	wrapped := &GenAIAgent{
		appName: appName,
		agent:   baseAgent,
	}
	return &GenAIStructuredAgent[TReq, TRes]{
		base:      wrapped,
		outputKey: outputKey,
	}, nil
}
func (a *GenAIStructuredAgent[TReq, TRes]) NewInMemorySession(
	ctx context.Context,
	userID string,
) GenAIStructuredSessionInterface[TReq, TRes] {
	baseSession := a.base.NewInMemorySession(ctx, userID)
	return &GenAIStructuredSession[TReq, TRes]{
		base:      baseSession,
		outputKey: a.outputKey,
	}
}

func (a *GenAIStructuredAgent[TReq, TRes]) NewVertexSession(
	ctx context.Context,
	userID string,
) (GenAIStructuredSessionInterface[TReq, TRes], error) {
	baseSession, err := a.base.NewVertexSession(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("error creating vertix session: %w", err)
	}
	return &GenAIStructuredSession[TReq, TRes]{
		base:      baseSession,
		outputKey: a.outputKey,
	}, nil
}

func (a *GenAIStructuredAgent[TReq, TRes]) NewRedisSession(
	ctx context.Context,
	userID string,
	sessionID string,
	rdb *redis.Client,
) (GenAIStructuredSessionInterface[TReq, TRes], error) {
	baseSession, err := a.base.NewRedisSession(ctx, userID, sessionID, rdb)
	if err != nil {
		return nil, err
	}
	return &GenAIStructuredSession[TReq, TRes]{
		base:      baseSession,
		outputKey: a.outputKey,
	}, nil
}
