# genaiclient 🧠

**A modular Go SDK for building AI Agents, Tools, and Chats — powered by Google GenAI and Redis.**

---

## 🚀 Overview

`genaiclient` is a lightweight, extensible Go library that abstracts LLM agent and chat management into clean, composable layers.

It lets you:

- Create and persist **AI Agents** with custom personas and tools.
- Manage **Chat sessions** and histories via Redis.
- Store **User contexts** for personalized conversations.
- Extend agents with **Go functions** automatically exposed as GenAI tools.
- Embed text using model APIs.

This SDK aims to simplify building **multi-agent systems**, **LLM microservices**, or **AI assistants** in production.

---

## 🧩 Architecture

```

┌────────────────────────────────────────┐
│             Genaiclient                │
│ Main entrypoint – factory + stateless  │
│  • NewAgent / GetAgent / ListAgents    │
│  • Embed / EmbedBulk                   │
└────────────────────────────────────────┘
│
▼
┌────────────────────────────────────────┐
│               Agent                    │
│ Stateful AI persona w/ tools & chats   │
│  • AddTool / RemoveTool                │
│  • GenerateWithContext                 │
│  • NewChat / GetChat / ListChats       │
└────────────────────────────────────────┘
│
▼
┌────────────────────────────────────────┐
│                Chat                    │
│ Stateful conversation per user         │
│  • GetHistory                          │
│  • SendMessage / SendMessageStream     │
└────────────────────────────────────────┘
│
▼
┌────────────────────────────────────────┐
│             RedisClient                │
│ Data Access Layer for persistence      │
│  • Agent / Chat / User CRUD            │
│  • SaveChatMessage, GetChatHistory     │
└────────────────────────────────────────┘

```

---

## 📦 Installation

```bash
go get github.com/darwishdev/genaiclient
```

---

## 🧰 Quick Start

```go
package main

import (
    "context"
    "fmt"

    "github.com/darwishdev/genaiclient"
    "github.com/darwishdev/genaiclient/pkg/genaiconfig"
    "github.com/redis/go-redis/v9"
    "google.golang.org/genai"
)

func main() {
    ctx := context.Background()

    // Initialize dependencies
    genaiClient, _ := genai.NewClient(ctx)
    redisInstance := redis.NewClient(&redis.Options{Addr: "localhost:6379"})

    // Initialize the library
    client, _ := genaiclient.NewGenaiClient(ctx, genaiClient, redisInstance)

    // Create a new agent
    agentConfig := genaiconfig.AgentConfig{
        ID:                "assistant-1",
        Persona:           "Friendly Assistant",
        SystemInstruction: "You are a helpful AI assistant.",
    }

    agent, _ := client.NewAgent(ctx, agentConfig)
    fmt.Println("Created agent:", agent.GetConfig().Persona)

    // Start a chat
    chatConfig := genaiconfig.ChatConfig{ID: "chat-001"}
    chat, _ := agent.NewChat(ctx, "user-123", chatConfig)

    // Send a message
    prompt := genaiconfig.Prompt{Text: "Hello there!"}
    response, _ := chat.SendMessage(ctx, prompt)

    fmt.Println("Model response:", response)
}
```

---

## ⚙️ Core Concepts

### **1. Agents**

An Agent represents a persona or model configuration, e.g., “Data Analyst” or “Code Helper.”
Agents are persisted in Redis and can have tools attached dynamically.

```go
agent.AddTool(ctx, MySummarizeFunction)
agent.RemoveTool(ctx, "MySummarizeFunction")
```

---

### **2. Chats**

Chats are user-specific, stateful conversations managed via Redis.

```go
chat, _ := agent.NewChat(ctx, "user-42", genaiconfig.ChatConfig{ID: "c-42"})
chat.SendMessage(ctx, genaiconfig.Prompt{Text: "Summarize this file."})
```

---

### **3. Tools**

Tools are Go functions that are auto-converted to GenAI `Tool` schemas.

```go
func TranslateText(source, target, text string) (string, error) {
    // Implementation
}
```

Registered via:

```go
agent.AddTool(ctx, TranslateText)
```

---

### **4. Embeddings**

Use `Embed()` or `EmbedBulk()` for semantic vector generation.

```go
vector, _ := client.Embed(ctx, "semantic meaning of life")
```

---

## 🧱 Redis Schema (Recommended)

| Key Pattern         | Description                 |
| ------------------- | --------------------------- |
| `agent:<id>`        | JSON of `AgentConfig`       |
| `chat:<id>`         | JSON of `ChatConfig`        |
| `chat:<id>:history` | List of `ChatMessage` JSONs |
| `user:<id>`         | JSON of `User` context      |

---

## 🧪 Testing

All interfaces are mockable. Example using GoMock:

```go
mockRedis := NewMockRedisClientInterface(ctrl)
mockRedis.EXPECT().CreateAgent(gomock.Any(), gomock.Any()).Return(nil)
```

---

## 🧭 Roadmap

- [ ] Implement Redis persistence methods.
- [ ] Integrate GenAI embedding APIs.
- [ ] Add adapter reflection for tool schema generation.
- [ ] Add streaming support in `Chat.SendMessageStream`.
- [ ] Optional persistence backends (PostgreSQL, Firestore).
- [ ] Add example multi-agent orchestrator.

---

## 📄 License

MIT © 2025 [darwishdev](https://github.com/darwishdev)

---

## 🤝 Contributing

Contributions welcome!
Open issues or PRs with clear reproduction steps and context.

```

```

