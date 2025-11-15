package genaiclient

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

type GenAIAgentInterface interface {
	NewInMemorySession(ctx context.Context, userID string) GenAISessionInterface
	NewVertexSession(ctx context.Context, userID string) (GenAISessionInterface, error)
	NewRedisSession(ctx context.Context, userID string, sessionID string, rdb *redis.Client) (GenAISessionInterface, error)
}
type GenAIAgent struct {
	model                *model.LLM
	agent                agent.Agent
	appName              string
	userID               string
	modelName            string
	sessionService       session.Service
	beforeModelCallbacks []llmagent.BeforeModelCallback
	afterModelCallbacks  []llmagent.AfterModelCallback
	tracerEnabled        bool
}

func EnableTracer() (llmagent.BeforeModelCallback, llmagent.AfterModelCallback) {
	before := func(ctx agent.CallbackContext, llmRequest *model.LLMRequest) (*model.LLMResponse, error) {
		for i, content := range llmRequest.Contents {
			log.Debug().
				Int("index", i).
				Interface("role", content.Role).
				Interface("parts", content.Parts).
				Msg("Before LLM call - content")
		}

		// Log session state in a safe way
		ctx.State().All()(func(k string, v any) bool {
			log.Debug().Str("key", k).Interface("value", v).Msg("state (before)")
			return true
		})

		return nil, nil
	}

	after := func(ctx agent.CallbackContext, llmResponse *model.LLMResponse, llmErr error) (*model.LLMResponse, error) {
		if llmResponse != nil && llmResponse.Content != nil {
			log.Debug().
				Interface("role", llmResponse.Content.Role).
				Interface("parts", llmResponse.Content.Parts).
				Msg("After LLM call - content")
		}
		if llmErr != nil {
			log.Error().Err(llmErr).Msg("LLM returned error")
		}
		ctx.State().All()(func(k string, v any) bool {
			log.Debug().Str("key", k).Interface("value", v).Msg("state (after)")
			return true
		})

		return nil, nil
	}
	return before, after
}

func NewGeminiAgent(appName string,
	apiKey string,
	modelName string,
	agentName string,
	agentDescription string,
	agentInstructions string,
	beforeModelCallbacks []llmagent.BeforeModelCallback,
	afterModelCallbacks []llmagent.AfterModelCallback,
	enableTracer bool,
	overridConfig ...llmagent.Config,
) (GenAIAgentInterface, error) {
	if enableTracer {
		b, a := EnableTracer()
		beforeModelCallbacks = append(beforeModelCallbacks, b)
		afterModelCallbacks = append(afterModelCallbacks, a)
	}
	ctx := context.Background()
	model, err := gemini.NewModel(ctx, modelName, &genai.ClientConfig{APIKey: apiKey})
	if err != nil {
		return nil, fmt.Errorf("Failed to create model: %w", err)
	}
	var finalCfg llmagent.Config
	if len(overridConfig) > 0 {
		finalCfg = overridConfig[0]
	} else {
		finalCfg = llmagent.Config{}
	}
	finalCfg.Name = agentName
	finalCfg.Model = model
	finalCfg.Description = agentDescription
	finalCfg.BeforeModelCallbacks = beforeModelCallbacks
	finalCfg.AfterModelCallbacks = afterModelCallbacks
	finalCfg.Instruction = agentInstructions
	agent, err := llmagent.New(finalCfg)
	if err != nil {
		return nil, fmt.Errorf("Failed to create agent: %w", err)
	}
	return &GenAIAgent{
		appName:              appName,
		modelName:            modelName,
		agent:                agent,
		beforeModelCallbacks: beforeModelCallbacks,
		afterModelCallbacks:  afterModelCallbacks,
	}, nil
}
func NewGenAIAgentFromConfig(appName string, cfg llmagent.Config, enableTracer bool) (GenAIAgentInterface, error) {
	if enableTracer {
		before, after := EnableTracer()
		cfg.BeforeModelCallbacks = append(cfg.BeforeModelCallbacks, before)
		cfg.AfterModelCallbacks = append(cfg.AfterModelCallbacks, after)
	}
	agent, err := llmagent.New(cfg)
	if err != nil {
		return nil, err
	}
	return &GenAIAgent{
		appName: appName,
		agent:   agent,
	}, nil
}
func (a *GenAIAgent) traceEvent(ev *session.Event) {
	if !a.tracerEnabled || ev == nil {
		return
	}
	log.Debug().
		Str("app", a.appName).
		Str("user", a.userID).
		Str("author", ev.Author).
		Interface("content", ev.LLMResponse.Content).
		Msg("Agent event")
}
func (a *GenAIAgent) NewInMemorySession(ctx context.Context, userID string) GenAISessionInterface {
	sessionService := session.InMemoryService()
	sessionCreateResponse, err := sessionService.Create(ctx, &session.CreateRequest{
		AppName: a.appName,
		UserID:  userID,
	})
	if err != nil {
		panic(err)
	}

	session := sessionCreateResponse.Session
	config := runner.Config{
		AppName:        a.appName,
		Agent:          a.agent,
		SessionService: sessionService,
	}
	runner, err := runner.New(config)
	if err != nil {
		panic(err)
	}
	return &GenAISession{
		session: session,
		runner:  runner,
	}
}

func (a *GenAIAgent) NewVertexSession(ctx context.Context, userID string) (GenAISessionInterface, error) {
	vertexService, err := session.VertexAIService(ctx, a.modelName)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vertex AI session service: %w", err)
	}

	// 2. Create the session through the Vertex AI service
	sessionResp, err := vertexService.Create(ctx, &session.CreateRequest{
		AppName: a.appName,
		UserID:  userID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Vertex AI session: %w", err)
	}

	session := sessionResp.Session
	// 3. Configure the runner
	config := runner.Config{
		AppName:        a.appName,
		Agent:          a.agent,
		SessionService: vertexService,
	}

	runnerInstance, err := runner.New(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create runner: %w", err)
	}

	return &GenAISession{
		session: session,
		runner:  runnerInstance,
	}, nil
}

func (a *GenAIAgent) NewRedisSession(ctx context.Context, userID string, sessionID string, rdb *redis.Client) (GenAISessionInterface, error) {
	redisService := NewRedisSessionService(rdb, 0)
	sessionResp, err := redisService.Create(ctx, &session.CreateRequest{
		AppName:   a.appName,
		SessionID: sessionID,
		UserID:    userID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis session: %w", err)
	}
	config := runner.Config{
		AppName:        a.appName,
		Agent:          a.agent,
		SessionService: redisService,
	}
	runnerInstance, err := runner.New(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create runner: %w", err)
	}
	return &GenAISession{
		session: sessionResp.Session,
		runner:  runnerInstance,
	}, nil
}
