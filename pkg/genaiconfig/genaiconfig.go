package genaiconfig

import "google.golang.org/genai"

type FunctionCallingMode string

const (
	// FunctionCallingModeUnspecified - The function calling config mode is unspecified
	FunctionCallingModeUnspecified FunctionCallingMode = "MODE_UNSPECIFIED"
	// FunctionCallingModeAuto - Default model behavior, model decides to predict either function calls or natural language response
	FunctionCallingModeAuto FunctionCallingMode = "AUTO"
	// FunctionCallingModeAny - Model is constrained to always predicting function calls only
	FunctionCallingModeAny FunctionCallingMode = "ANY"
	// FunctionCallingModeNone - Model will not predict any function calls
	FunctionCallingModeNone FunctionCallingMode = "NONE"
	// FunctionCallingModeValidated - Model decides but validates function calls with constrained decoding
	FunctionCallingModeValidated FunctionCallingMode = "VALIDATED"
)

type SchemaConfig struct {
	// Optional. Describes the parameters to the function in JSON Schema format. The schema
	// must describe an object where the properties are the parameters to the function.
	// For example: ``` { "type": "object", "properties": { "name": { "type": "string" },
	SchemaJSON map[string]interface{} // json represnetaion of the genai schema on a hash map
	// Optional. Describes the parameters to this function in JSON Schema Object format.
	// Reflects the Open API 3.03 Parameter Object. string Key: the name of the parameter.
	// Parameter names are case sensitive. Schema Value: the Schema defining the type used
	// for the parameter. For function with no parameters, this can be left unset. Parameter
	// names must start with a letter or an underscore and must only contain chars a-z,
	// A-Z, 0-9, or underscores with a maximum length of 64. Example with 1 required and
	// 1 optional parameter: type: OBJECT properties: param1: type: STRING param2: type:
	// INTEGER required: - param1
	SchemaGenAI *genai.Schema
	// struct type to infer the schema from it via the reflect and json annotations
	Schema any
}
type Tool struct {
	Name           string
	Description    string
	RequestConfig  *SchemaConfig
	ResponseConfig *SchemaConfig
}

type ToolConfig struct {
	Mode         FunctionCallingMode `json:"mode"`
	AllowedTools []string            `json:"allowedTools,omitempty"`
}

type ChatType string

const (
	ChatTypeConversational ChatType = "CONVERSATIONAL"
	ChatTypeBackground     ChatType = "BACKGROUND"
)

type AgentConfig struct {
	ID                      string            `json:"id"`
	Persona                 string            `json:"persona"`
	SystemInstruction       string            `json:"systemInstruction"`
	DefaultModel            string            `json:"deaultModel"`
	DefaultGenerationConfig *GenerationConfig `json:"defaultGenerationConfig"`
}
type User struct {
	ID      string
	Context string
}
type FileConfig struct {
	Path     string `json:"path"` // could be local or remote file
	Contents []byte // could be passed directly and in this case we will totally ignore the path
	Name     string `json:"name,omitempty"`
	Context  string
	MIMEType string                 `json:"mimeType,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}
type Prompt struct {
	Text           string
	StructuredText map[string]interface{}
	Files          []FileConfig // could be audio or files shjould be infered from the mimeType
	Model          string
}
type ChatConfig struct {
	ID               string `json:"id"`
	AgentID          string `json:"agentID"`
	UserID           string `json:"userID"`
	Model            string
	GenerationConfig *GenerationConfig `json:"generationConfig,omitempty"`
	Type             ChatType          `json:"type"`
}

// GenerationConfig provides a comprehensive control panel for all generation requests.
type GenerationConfig struct {
	Temperature          *float32 `json:"temperature,omitempty"`
	TopP                 *float32 `json:"topP,omitempty"`
	TopK                 *float32 `json:"topK,omitempty"`
	MaxOutputTokens      int32    `json:"maxOutputTokens,omitempty"`
	StopSequences        []string `json:"stopSequences,omitempty"`
	ResponseSchemaConfig *SchemaConfig
	Tools                []*Tool     `json:"tools,omitempty"`
	ToolConfig           *ToolConfig `json:"toolConfig,omitempty"`
}
type ModelResponse struct {
	Text         string
	FunctionCall *FunctionCall
	Error        error
}
type FunctionCall struct {
	Name string
	Args map[string]interface{}
}
type ChatMessage struct {
	Role    string `json:"role"` // "user", "model", or "tool"
	Content string `json:"content"`
}
