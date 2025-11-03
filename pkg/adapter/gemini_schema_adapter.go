package adapter

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/darwishdev/genaiclient/pkg/genaiconfig"
	genai "google.golang.org/genai"
)

type Type string

const (
	TypeString  Type = "STRING"
	TypeInteger Type = "INTEGER"
	TypeNumber  Type = "NUMBER"
	TypeBoolean Type = "BOOLEAN"
	TypeObject  Type = "OBJECT"
	TypeArray   Type = "ARRAY"
)

func BuildSchemaFromJson(v []byte) (*genai.Schema, error) {
	var genSchema genai.Schema
	err := json.Unmarshal(v, &genSchema)
	if err != nil {
		return nil, fmt.Errorf("❌ getting schema from json failed: %w", err)
	}
	return &genSchema, nil
}

func buildSchemaFromType(t reflect.Type) *genai.Schema {
	s := &genai.Schema{}

	switch t.Kind() {
	case reflect.Struct:
		s.Type = genai.TypeObject
		s.Properties = map[string]*genai.Schema{}

		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.PkgPath != "" { // skip unexportede
				continue
			}

			jsonTag := f.Tag.Get("json")
			parts := strings.Split(jsonTag, ",")
			fieldName := parts[0]
			if fieldName == "" {
				fieldName = f.Name
			}

			fieldSchema := buildSchemaFromType(baseType(f.Type))
			s.Properties[fieldName] = fieldSchema
			isOmitempty := false
			for _, opt := range parts[1:] {
				if opt == "omitempty" {
					isOmitempty = true
					break
				}
			}

			// Only append to s.Required if 'omitempty' is NOT found.
			if !isOmitempty {
				s.Required = append(s.Required, fieldName)
			}
		}

	case reflect.Slice, reflect.Array:
		s.Type = genai.TypeArray
		s.Items = buildSchemaFromType(baseType(t.Elem()))

	case reflect.String:
		s.Type = genai.TypeString

	case reflect.Bool:
		s.Type = genai.TypeBoolean

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		s.Type = genai.TypeInteger

	case reflect.Float32, reflect.Float64:
		s.Type = genai.TypeNumber

	default:
		s.Type = genai.TypeString
	}

	return s
}
func NewToolFromSignatures[TReq, TRes any](
	name string,
	description string,
	reqSignature TReq,
	resSignature TRes,
) (genaiconfig.Tool, error) {
	reqRef := reflect.TypeOf(reqSignature)
	// --- 1. Process Request Schema ---
	reqSchema := buildSchemaFromType(reqRef)

	resRef := reflect.TypeOf(resSignature)
	// --- 2. Process Response Schema ---
	resSchema := buildSchemaFromType(resRef)
	// --- 3. Assemble the Tool ---
	return genaiconfig.Tool{
		Name:        name,
		Description: description,
		RequestConfig: &genaiconfig.SchemaConfig{
			// Note: RequestSchemaJSON should be derived from reqSchema, e.g., via a helper function
			SchemaGenAI: reqSchema,
		},
		ResponseConfig: &genaiconfig.SchemaConfig{
			// Note: ResponseSchemaJSON should be derived from resSchema
			SchemaGenAI: resSchema,
		},
	}, nil
}
func baseType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}
func float32Ptr(v float32) *float32 { return &v }
