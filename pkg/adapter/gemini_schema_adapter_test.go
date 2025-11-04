package adapter

import (
	"encoding/json" // ⬅️ NEW: Required for JSON marshaling
	"reflect"
	"testing"

	"google.golang.org/genai"
	// NOTE: If genaiconfig.Tool is used in TestNewToolFromSignatures, you'd need the import here.
	// We'll rely on the MockTool for JSON marshaling in that test.
)

// Mock/Test Structs for Schema Generation (Remain unchanged)
type NestedStruct struct {
	ID      int  `json:"id"`
	IsReady bool `json:"is_ready"`
}

type ComplexRequest struct {
	Name    string        `json:"user_name,omitempty"`
	Age     int32         `json:"user_age,omitempty"`
	Score   float64       `json:"score,omitempty"`
	Items   []string      `json:"items"`
	Details NestedStruct  `json:"details"`
	Pointer *NestedStruct `json:"pointer_details"`
	Array   [2]int        `json:"fixed_array"`
}

type SimpleResponse struct {
	Status string `json:"status"`
}

// NOTE: The actual `buildSchemaFromType` function is assumed to exist in the adapter package.

// --- Helper function from previous context (for clarity, not repeated here) ---
// func buildSchemaFromType(t reflect.Type) *genai.Schema { ... }
// func baseType(t reflect.Type) reflect.Type { ... }
// func NewToolFromSignatures[TReq, TRes any](...) (genaiconfig.Tool, error) { ... }

func Test_buildSchemaFromType(t *testing.T) {
	// A basic function to check if a schema property matches expectations
	checkProp := func(t *testing.T, schema *genai.Schema, name string, expectedType genai.Type) {
		prop, ok := schema.Properties[name]
		if !ok {
			t.Errorf("Property %s not found in schema", name)
			return
		}
		if prop.Type != expectedType {
			t.Errorf("Property %s type is %v, want %v", name, prop.Type, expectedType)
		}
	}

	tests := []struct {
		name         string
		input        any
		expectedType genai.Type
		check        func(*testing.T, *genai.Schema)
	}{
		{
			name:         "Complex Struct Request",
			input:        ComplexRequest{},
			expectedType: genai.TypeObject,
			check: func(t *testing.T, s *genai.Schema) {
				if s.Type != genai.TypeObject {
					t.Fatalf("Expected TypeObject, got %v", s.Type)
				}
				if len(s.Required) != 4 {
					t.Errorf("Expected 7 required fields, got %d", len(s.Required))
				}
				checkProp(t, s, "user_name", genai.TypeString)
				checkProp(t, s, "user_age", genai.TypeInteger)
				checkProp(t, s, "score", genai.TypeNumber)
				// ... (rest of property checks omitted for brevity)
			},
		},
		{
			name:         "Simple String Response",
			input:        SimpleResponse{},
			expectedType: genai.TypeObject,
			check: func(t *testing.T, s *genai.Schema) {
				checkProp(t, s, "status", genai.TypeString)
			},
		},
		{
			name:         "Primitive String",
			input:        "",
			expectedType: genai.TypeString,
			check: func(t *testing.T, s *genai.Schema) {
				if s.Type != genai.TypeString {
					t.Errorf("Expected TypeString, got %v", s.Type)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("--- STARTING TEST CASE: %s ---", tt.name)

			// Simulate the reflection logic and log the input
			inputType := reflect.TypeOf(tt.input)
			t.Logf("INPUT TYPE: %s (%v)", inputType.Name(), inputType.Kind())

			// Call the function under test
			schema := buildSchemaFromType(inputType)

			// 🚀 FIX: Marshal the schema to indented JSON for human-readable output
			jsonBytes, err := json.MarshalIndent(schema, "", "  ")
			if err != nil {
				t.Fatalf("Failed to marshal schema to JSON for logging: %v", err)
			}

			t.Logf("GENERATED SCHEMA (Type): %v", schema.Type)
			t.Logf("GENERATED SCHEMA (Required Fields): %v", schema.Required)
			t.Logf("\n--- GENERATED SCHEMA (JSON) ---\n%s\n--- END JSON ---\n", jsonBytes) // Log the JSON

			// Continue with assertions
			if schema.Type != tt.expectedType {
				t.Fatalf("buildSchemaFromType() type = %v, want %v", schema.Type, tt.expectedType)
			}
			tt.check(t, schema)

			t.Logf("--- ENDING TEST CASE: %s (PASS) ---", tt.name)
		})
	}
}

// // Define a minimal mock for the external type
type MockToolRequestConfig struct {
	RequestSchemaGenAI *genai.Schema
}
type MockToolResponseConfig struct {
	ResponseSchemaGenAI *genai.Schema
}
type MockTool struct {
	Name           string
	Description    string
	RequestConfig  MockToolRequestConfig
	ResponseConfig MockToolResponseConfig
}

func TestNewToolFromSignatures(t *testing.T) {
	t.Log("--- STARTING TestNewToolFromSignatures ---")

	req := ComplexRequest{}
	res := SimpleResponse{}
	toolName := "test_tool"
	toolDesc := "A tool for testing."

	t.Logf("TOOL INPUT: Name='%s', Description='%s'", toolName, toolDesc)
	t.Logf("REQUEST SIGNATURE: %s", reflect.TypeOf(req).Name())
	t.Logf("RESPONSE SIGNATURE: %s", reflect.TypeOf(res).Name())

	// Call the function under test
	tool, err := NewToolFromSignatures(toolName, toolDesc, req, res) // Returns genaiconfig.Tool, assumed similar to MockTool

	if err != nil {
		t.Fatalf("NewToolFromSignatures() error = %v, want nil", err)
	}

	// 🚀 FIX: Marshal the final assembled Tool object to indented JSON.
	jsonBytes, jsonErr := json.MarshalIndent(tool, "", "  ")
	if jsonErr != nil {
		t.Fatalf("Failed to marshal final tool to JSON for logging: %v", jsonErr)
	}

	t.Logf("GENERATED TOOL NAME: %s", tool.Name)
	t.Logf("GENERATED TOOL DESCRIPTION: %s", tool.Description)
	t.Logf("REQUEST SCHEMA PROPERTIES COUNT: %d", len(tool.RequestConfig.SchemaGenAI.Properties))
	t.Logf("RESPONSE SCHEMA PROPERTIES COUNT: %d", len(tool.ResponseConfig.SchemaGenAI.Properties))
	t.Logf("\n--- GENERATED TOOL (JSON STRUCTURE) ---\n%s\n--- END JSON ---\n", jsonBytes) // Log the JSON

	// Continue with assertions
	if tool.Name != toolName {
		t.Errorf("Tool Name got = %v, want %v", tool.Name, toolName)
	}
	if tool.Description != toolDesc {
		t.Errorf("Tool Description got = %v, want %v", tool.Description, toolDesc)
	}

	// Check Request Schema
	reqSchema := tool.RequestConfig.SchemaGenAI
	if len(reqSchema.Properties) != 7 {
		t.Errorf("Request Schema expected 7 properties, got %d", len(reqSchema.Properties))
	}

	// Check Response Schema
	resSchema := tool.ResponseConfig.SchemaGenAI
	if len(resSchema.Properties) != 1 {
		t.Errorf("Response Schema expected 1 property, got %d", len(resSchema.Properties))
	}

	t.Log("--- ENDING TestNewToolFromSignatures (PASS) ---")
}
