package genaiconfig

type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type ToolConfig struct {
	Mode         string   `json:"mode"`
	AllowedTools []string `json:"allowedTools,omitempty"`
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
	DefaultGenerationConfig *GenerationConfig `json:"defaultGenerationConfig"`
	Tools                   []*Tool           `json:"tools"`
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
}
type ChatConfig struct {
	ID   string   `json:"id"`
	Type ChatType `json:"type"`
}

// GenerationConfig provides a comprehensive control panel for all generation requests.
type GenerationConfig struct {
	Temperature        *float32               `json:"temperature,omitempty"`
	TopP               *float32               `json:"topP,omitempty"`
	TopK               *int32                 `json:"topK,omitempty"`
	MaxOutputTokens    *int32                 `json:"maxOutputTokens,omitempty"`
	StopSequences      []string               `json:"stopSequences,omitempty"`
	ResponseJSONSchema map[string]interface{} `json:"responseJsonSchema,omitempty"`
	Tools              []*Tool                `json:"tools,omitempty"`
	ToolConfig         *ToolConfig            `json:"toolConfig,omitempty"`
}
type ModelResponse struct {
	Text         string
	FunctionCall *FunctionCall
}
type FunctionCall struct {
	Name string
	Args map[string]interface{}
}
type ChatMessage struct {
	Role    string `json:"role"` // "user", "model", or "tool"
	Content string `json:"content"`
}
