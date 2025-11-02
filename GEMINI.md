## I. Project Goal (Wrapper Library)

The purpose of this project is to create a Go package that acts as a simplified, ready-to-use **wrapper** around the `google.golang.org/genai` library's embedding functionality. The target audience for this wrapper is Go developers who want to perform common embedding tasks without managing the raw API request structures.

The wrapper must provide **specific functions** for the common use cases outlined in the documentation below.

## II. Agent Persona & Constraints

1.  **Persona & Role:** Act as a **Solution Architect** specializing in Go microservices and the `google.golang.org/genai` library. Your primary task is to **design and plan** the package structure and public interfaces. **Think first, code second.**
2.  **Architectural Approach:** Before writing any code, you must first output a detailed **Architectural Design Plan**. This plan must outline the following:
    - The proposed **struct** definition for the wrapper (e.g., `EmbedderService`), including any necessary client fields.
    - The **public function signatures** (names, inputs, outputs, and documentation comments) for the required wrapper functions.
    - A brief justification for the chosen design patterns (e.g., dependency injection for the Gemini client).
3.  **Naming:** All public structs and functions must start with **`Embedder`** (e.g., `NewEmbedder`, `EmbedderSingleText`).
4.  **Error Handling:** Always include robust error handling and return Go errors where appropriate.
5.  **Structure:** The main package file is `embedder.go`.

## III. Embedded Documentation (Go Embeddings API)

The agent must use the following official documentation for all code generation:

### A. Generating a Single Embedding

To generate an embedding for one piece of content, use `client.Models.EmbedContent`.

- **Model:** `"gemini-embedding-001"` (must be used for all embedding calls).
- **Input:** `[]*genai.Content`
- **Method:** `client.Models.EmbedContent(ctx, modelName, contents, config)`

**Go Example:**

```go
client, err := genai.NewClient(ctx, nil)
// ... error handling
contents := []*genai.Content{
    genai.NewContentFromText("What is the meaning of life?", genai.RoleUser),
}
result, err := client.Models.EmbedContent(ctx, "gemini-embedding-001", contents, nil)
// ... result.Embeddings[0].Values contains the embedding
```
