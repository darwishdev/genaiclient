# Go GenAI Client: An Agentic Framework for Google's Gemini

Go GenAI Client is a production-ready, agentic framework for Go that provides a robust, scalable, and developer-friendly interface for building applications on top of Google's Generative AI (Gemini) models.

This package moves beyond a simple API wrapper, offering a powerful, stateful architecture centered around the concept of **AI Agents**. It is designed for building complex, user-centric applications that require persistent memory, dynamic capabilities, and intelligent caching.

## Core Principles

*   **Agent-Centric:** The `Agent` is the primary entity. It encapsulates a persona, capabilities (tools), and default behaviors, making it a reusable and configurable actor in your system.
*   **Persistence by Default:** Built-in Redis integration ensures that agent configurations and user chat histories are stateful and persistent, enabling long-term memory and scalable deployments.
*   **User-Aware:** The end-user is a first-class citizen. All interactions are mapped to a `userID`, providing a clear path to building multi-tenant applications and retrieving user-specific history.
*   **Configuration over Code:** Define *what* you want an agent to do via rich configuration objects, not by writing boilerplate. The framework handles the complex "how."

## Features (The Storybook)

Here’s what you can do with Go GenAI Client:

*   **Agent Management:**
    *   As a developer, I can **create a new AI Agent** with a specific persona, system instructions, and default generation settings (e.g., temperature, response format).
    *   As a developer, I can **list all available Agents** to see what capabilities are available in my application.
    *   As a developer, I can **retrieve an Agent by its ID** to use it for a task.
    *   As a developer, I can **delete an Agent** that is no longer needed.

*   **Dynamic Capabilities (Tool Use):**
    *   As a developer, I can define a standard Go function and **dynamically add it as a Tool** to an existing Agent.
    *   As a developer, the framework will **automatically convert my Go function signature into a JSON Schema** that the Gemini model can understand.
    *   As a developer, I can **remove a Tool** from an Agent when its capability is no longer required.

*   **User-Centric Conversations:**
    *   As a developer, I can **start a new Chat session** between a specific `userID` and an `Agent`.
    *   As a developer, I can create a **background chat**, a hidden, programmatic chat that maintains user-specific history but doesn't appear in the user's main chat list.
    *   As a developer, I can **retrieve the full conversation history** for a specific user, with the option to include or exclude background chats.    *   As a developer, I can **load a previous Chat session** to continue a conversation.
    *   As a developer, I can ask an Agent to **generate a response with automatic context**. The framework will automatically fetch the user's recent chat history and include it in the prompt.

*   **Advanced Generation & Caching:**
    *   As a developer, I can request a **structured response (JSON)** from the model by providing a Go struct, and the framework will handle the schema generation and prompt engineering.
    *   As a developer, I can implement **intelligent caching strategies**. I can enable caching for a request, set a custom TTL, or force a cache refresh, all on a per-call basis.
    *   As a developer, all my interactions can be **streamed**, allowing for real-time, responsive applications.

*   **Stateless Utilities:**
    *   As a developer, I can still perform simple, stateless tasks like **generating text embeddings** for semantic search or other use cases.

## Project Structure

This project follows a standard Go library structure that separates the public API from the internal implementation.

*   `/` **(Root Package)**: Defines all the public interfaces (`Genaiclient`, `Agent`, `Chat`, etc.). This is the contract that your application will consume.
*   `app/`: Contains the concrete implementation of the interfaces defined in the root package. Your application will use this package to create a new client instance.
*   `pkg/`: Contains shared internal code, such as the `adapter` functions that translate between this library's types and the underlying `google.golang.org/genai` types.

## Getting Started

### Prerequisites

*   Go 1.18+
*   An active Google AI API Key.
*   A running Redis instance.

### Installation

```sh
go get github.com/darwishdev/genaiclient
```

### Example Usage

Here is a complete example demonstrating how to use this package as a library to create a "Weather Reporter" agent and interact with it, showcasing structured responses, background chats, and smart caching.

```go
package myapplication

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"google.golang.org/api/option"

	// Import the interfaces from the root package
	"github.com/your-username/genaiclient"
	// Import the implementation from the app package to create a client
	"github.com/your-username/genaiclient/app"
	// Import the adapter for schema generation
	"github.com/your-username/genaiclient/pkg/adapter"
)

// 1. Define a Go function to be used as a tool.
// The comments are important, as they become the description for the model.
func getCurrentWeather(city string) (string, error) {
	// In a real application, this would call a weather API.
	if city == "New York" {
		return `{"temperature": "15°C", "conditions": "Cloudy"}`, nil
	}
	return `{"temperature": "unknown", "conditions": "unknown"}`, nil
}

// 2. Define a struct for structured responses (e.g., a job application form).
type JobApplicationForm struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	Position    string `json:"position"`
	Experience  int    `json:"experience"`
	CoverLetter string `json:"coverLetter"`
}

func RunExample() {
	ctx := context.Background()

	// --- Setup ---

googleAIOpts := option.WithAPIKey("YOUR_GOOGLE_AI_API_KEY")
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	client, err := app.NewClient(ctx, googleAIOpts, redisClient)
	if err != nil {
		panic(err)
	}

	// --- Agent Creation (Weather Reporter) ---
	agentConfig := genaiclient.AgentConfig{
		ID:      "weather-reporter-v1",
		Persona: "You are a friendly weather reporter. You must use the tools provided to answer questions about the weather.",
	}

	agent, err := client.Agents().CreateAgent(ctx, agentConfig)
	if err != nil {
		// Handle error (e.g., agent might already exist)
		agent, _ = client.Agents().GetAgent(ctx, "weather-reporter-v1")
	}

	if err := agent.AddTool(ctx, getCurrentWeather); err != nil {
		panic(err)
	}

	fmt.Printf("Agent '%s' (Weather Reporter) is ready.\n", agent.ID())

	// --- User Interaction (Conversational Chat) ---
	const userID = "user-1234"
	chat, err := agent.StartChat(ctx, userID)
	if err != nil {
		panic(err)
	}

	prompt := "What's the weather like in New York?"
	response, err := chat.SendMessage(ctx, prompt)
	if err != nil {
		panic(err)
	}

	fmt.Printf("\nUser (Conversational): %s\n", prompt)
	fmt.Printf("Agent (Conversational): %s\n", response.Text)

	// --- Structured Response Agent (Form Filler) ---
	// Generate JSON schema from our Go struct
	formSchema, err := adapter.GeminiSchemaFromStruct(JobApplicationForm{})
	if err != nil {
		panic(err)
	}

	formFillerAgentConfig := genaiclient.AgentConfig{
		ID:      "form-filler-v1",
		Persona: "You are an expert form-filling assistant. You always respond with a JSON object matching the provided schema.",
		DefaultGenerationConfig: &genaiclient.GenerationConfig{
			ResponseJSONSchema: formSchema,
			Temperature:        float32Ptr(0.0), // For deterministic output
		},
	}
	formFillerAgent, err := client.Agents().CreateAgent(ctx, formFillerAgentConfig)
	if err != nil {
		formFillerAgent, _ = client.Agents().GetAgent(ctx, "form-filler-v1")
	}
	fmt.Printf("\nAgent '%s' (Form Filler) is ready.\n", formFillerAgent.ID())

	// --- Background Chat (Form Filling) ---
	// This chat will not appear in the user's default chat list.
	backgroundChat, err := formFillerAgent.StartChat(ctx, userID, genaiclient.StartChatOptions{
		Type: genaiclient.ChatTypeBackground,
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("\nStarted background chat (ID: %s) for user '%s'.\n", backgroundChat.ID(), userID)

	formPrompt := "Fill out a job application for John Doe, email john@example.com, for a Senior Software Engineer position with 10 years of experience. Write a short cover letter about his passion for Go."
	formResponse, err := backgroundChat.SendMessage(ctx, formPrompt)
	if err != nil {
		panic(err)
	}

	var filledForm JobApplicationForm
	if err := json.Unmarshal([]byte(formResponse.Text), &filledForm); err != nil {
		panic(err)
	}
	fmt.Printf("Background Chat (Form Filler) Output:\n%+v\n", filledForm)

	// --- Smart Caching Example ---
	// Let's ask the weather agent again, but force a cache refresh.
	fmt.Printf("\nUser (Conversational): What's the weather like in London? (with cache refresh)\n")
	cacheRefreshConfig := &genaiclient.GenerationConfig{
		CachingPolicy: &genaiclient.CachingPolicy{
			Enabled: true,
			Refresh: true,
			TTL:     1 * time.Minute,
		},
	}
	weatherResponseCached, err := chat.SendMessage(ctx, "What's the weather like in London?", cacheRefreshConfig)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Agent (Conversational, Cached): %s\n", weatherResponseCached.Text)

	// --- Listing Chats (excluding background by default) ---
	userManager := client.Users()
	userChats, err := userManager.ListChats(ctx, userID, 0, 10) // Default excludes background
	if err != nil {
		panic(err)
	}

	fmt.Printf("\n--- User '%s' Conversational Chats ---\n", userID)
	for _, c := range userChats {
		fmt.Printf("  Chat ID: %s, Agent ID: %s, Type: %s\n", c.ID(), c.AgentID(), c.Type())
	}

	// --- Listing Chats (including background) ---
	allUserChats, err := userManager.ListChats(ctx, userID, 0, 10, genaiclient.ListChatsOptions{IncludeBackground: true})
	if err != nil {
		panic(err)
	}

	fmt.Printf("\n--- User '%s' All Chats (including background) ---\n", userID)
	for _, c := range allUserChats {
		fmt.Printf("  Chat ID: %s, Agent ID: %s, Type: %s\n", c.ID(), c.AgentID(), c.Type())
	}
}

// Helper function for float32 pointers
func float32Ptr(f float32) *float32 { return &f }

// Helper function for int32 pointers
func int32Ptr(i int32) *int32 { return &i }
```

## API Reference

*(A brief overview of the main interfaces)*

The main interfaces are defined in the root `genaiclient` package. The concrete implementation and the `NewClient` constructor are located in the `app` package.

### `Genaiclient`
The main entry point for the library.
- `NewClient(ctx, googleAIOption, redisClient)`: Creates the client.
- `Agents() AgentManager`: Returns a manager for agent operations.
- `Users() UserManager`: Returns a manager for user-centric operations.

### `AgentManager`
Handles the lifecycle of agents.
- `CreateAgent(ctx, config)`: Creates and persists a new agent.
- `GetAgent(ctx, id)`: Retrieves an agent.
- `ListAgents(ctx, page, limit)`: Lists all agents.

### `UserManager`
Handles retrieval of user data.
- `ListChats(ctx, userID, page, limit, opts...)`: Gets all chats for a user, with options to include background chats.
- `GetMostRecentChat(ctx, userID, agentID...)`: Gets the last active chat for a user.

### `Agent`
The core interface for an AI actor.
- `ID()`, `Config()`: Accessors for agent data.
- `AddTool(ctx, goFunc)`, `RemoveTool(ctx, toolName)`: Manages agent capabilities.
- `StartChat(ctx, userID, opts...)`: Starts a new conversation with a user, with options to specify chat type.
- `GenerateWithContext(ctx, userID, prompt, ...)`: Generates a response with automatic history injection.

### `Chat`
Represents a single, stateful conversation.
- `ID()`, `UserID()`, `AgentID()`: Accessors for chat metadata.
- `Type()`: Returns the type of the chat (Conversational or Background).
- `SendMessage(ctx, prompt, ...)`: Sends a message and gets a response.
- `History(ctx)`: Returns the full message history for the chat.

---

This `README.md` provides a solid foundation for your project. You can now proceed with the implementation based on the final design we've crafted.