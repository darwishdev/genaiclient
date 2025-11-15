package genaiclient

import (
	"context"
	"iter"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

type GenAISessionInterface interface {
	Send(ctx context.Context, prompt string) iter.Seq2[*session.Event, error]
}

type GenAISession struct {
	session   session.Session
	outputKey string
	runner    *runner.Runner
}

func (s *GenAISession) Send(ctx context.Context, prompt string) iter.Seq2[*session.Event, error] {
	msg := &genai.Content{
		Parts: []*genai.Part{
			genai.NewPartFromText(prompt),
		},
		Role: string(genai.RoleUser),
	}
	cfg := agent.RunConfig{
		StreamingMode: agent.StreamingModeSSE,
	}
	itr := s.runner.Run(ctx, s.session.UserID(), s.session.ID(), msg, cfg)
	return itr
}
