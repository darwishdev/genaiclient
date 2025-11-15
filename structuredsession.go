package genaiclient

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"

	"google.golang.org/adk/session"
)

type GenAIStructuredSessionInterface[TReq any, TRes any] interface {
	Send(ctx context.Context, req TReq) (TRes, error)
	Handle(seq iter.Seq2[*session.Event, error]) (TRes, error)
}
type GenAIStructuredSession[TReq any, TRes any] struct {
	base      GenAISessionInterface
	outputKey string
}

func (s *GenAIStructuredSession[TReq, TRes]) Send(
	ctx context.Context,
	req TReq, // user passes structured request or string
) (TRes, error) {
	var prompt string
	if str, ok := any(req).(string); ok {
		prompt = str
	} else {
		b, err := json.Marshal(req)
		if err != nil {
			var zero TRes
			return zero, err
		}
		prompt = string(b)
	}
	seq := s.base.Send(ctx, prompt)
	return s.Handle(seq)
}

func (s *GenAIStructuredSession[TReq, TRes]) Handle(
	seq iter.Seq2[*session.Event, error],
) (TRes, error) {
	var out TRes
	var accumulated string
	for event, err := range seq {
		if err != nil {
			return out, fmt.Errorf("agent stream error: %w", err)
		}
		if event.Partial {
			for _, p := range event.Content.Parts {
				if p.Text != "" {
					accumulated += p.Text
					// Optionally print streaming text:
					fmt.Print(p.Text)
				}
			}
		}
	}
	if accumulated == "" {
		return out, fmt.Errorf("no response received")
	}
	if err := json.Unmarshal([]byte(accumulated), &out); err != nil {
		return out, fmt.Errorf("failed to parse structured response: %w", err)
	}
	return out, nil
}
