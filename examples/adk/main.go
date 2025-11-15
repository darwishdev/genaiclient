package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/darwishdev/genaiclient"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type CapitalRequest struct {
	Country string `json:"capital"`
}
type CapitalResponse struct {
	Capital string `json:"capital"`
}

func init() {
	// Enable colored console output
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	// Optional: include timestamps in human-readable format
	zerolog.TimeFieldFormat = "2006-01-02 15:04:05"
}
func structuredRedisAgentExample() {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // change if needed
		Password: "",               // no password
	})

	ctx := context.Background()
	err := rdb.Ping(ctx).Err()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to Redis")
	}

	log.Debug().Int("db", rdb.Options().DB).Msg("redis is up and connected to DB")

	// 2️⃣ Create structured agent
	agent, err := genaiclient.NewStructuredAgent[CapitalRequest, CapitalResponse](
		"my_app",
		"AIzaSyDDt0ZYks6oQHBdqapLRiM_kI7h2dAHsNU",
		"gemini-2.0-flash",
		"capital_redis_agent",
		"Provides the capital city of a country",
		"Respond with JSON containing {\"capital\": \"...\"}",
		true,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create structured agent")
	}

	// 3️⃣ Create Redis-backed session
	session, err := agent.NewRedisSession(ctx, "user_redis_2", "1", rdb)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create Redis session")
	}

	// 4️⃣ Send request
	response, err := session.Send(ctx, CapitalRequest{Country: "France"})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to send request")
	}

	// 5️⃣ Print response
	fmt.Printf("Response: %+v\n", response)

	response2, err := session.Send(ctx, CapitalRequest{Country: "Egypt"})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to send request")
	}

	// 5️⃣ Print response
	fmt.Printf("Response: %+v\n", response2)
	// Optional: wait a bit to simulate session persistence
	time.Sleep(time.Second)
}

func structuredVertexAgentExample() {
	// 1️⃣ Create a structured agent (same as before)
	agent, err := genaiclient.NewGeminiAgent(
		"my_app",
		"AIzaSyDDt0ZYks6oQHBdqapLRiM_kI7h2dAHsNU",
		"gemini-2.0-flash-lite",
		"capital_agent",
		"Provides the capital city of a country",
		"Respond with JSON containing {\"capital\": \"...\"}",
		nil,
		nil,
		true,
	)
	if err != nil {
		log.Debug().Err(err).Msg("error from gemini client")
	}
	session, err := agent.NewVertexSession(context.Background(), "user_1")
	if err != nil {
		log.Debug().Err(err).Msg("error from session")
	}
	response := session.Send(context.Background(), "Egypt")
	for event, err := range response {
		if err != nil {
			log.Debug().Err(err).Msg("error from llm")
			return
		}
		if event.Partial {
			for _, p := range event.Content.Parts {
				if p.Text != "" {
					fmt.Print(p.Text)
				}
			}
		}
	}
}
func structuredAgentExample() {
	agent, err := genaiclient.NewStructuredAgent[CapitalRequest, CapitalResponse](
		"my_app",
		"AIzaSyDDt0ZYks6oQHBdqapLRiM_kI7h2dAHsNU",
		"gemini-2.0-flash-lite",
		"capital_agent",
		"Provides the capital city of a country",
		"Respond with JSON containing {\"capital\": \"...\"}",
		true,
	)
	if err != nil {
		panic(err)
	}
	session := agent.NewInMemorySession(context.Background(), "user_1")
	response, err := session.Send(context.Background(), CapitalRequest{Country: "Egypt"})
	if err != nil {
		panic(err)
	}
	log.Debug().Interface("Response", response).Msg("Model")
}
func main() {
	structuredRedisAgentExample()
}
