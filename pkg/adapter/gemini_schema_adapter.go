package adapter

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
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

func BuildSchemaFromStruct[T interface{}](t T) *genai.Schema {
	return buildSchemaFromType(reflect.TypeOf(t))
}

func buildSchemaFromType(t reflect.Type) *genai.Schema {
	s := &genai.Schema{}

	switch t.Kind() {

	case reflect.Struct:
		s.Type = genai.TypeObject
		s.Properties = map[string]*genai.Schema{}
		s.PropertyOrdering = []string{}

		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)

			if f.PkgPath != "" { // skip unexported
				continue
			}

			jsonTag := f.Tag.Get("json")
			parts := strings.Split(jsonTag, ",")
			fieldName := parts[0]
			if fieldName == "" {
				fieldName = f.Name
			}

			fieldSchema := buildSchemaFromType(baseType(f.Type))

			// --- NEW: Read description tag ---
			if desc := f.Tag.Get("description"); desc != "" {
				fieldSchema.Description = desc
			}

			// --- NEW: Read minLength/maxLength ---
			if minLen := f.Tag.Get("minLength"); minLen != "" {
				if v, err := strconv.Atoi(minLen); err == nil {
					x := int64(v)
					fieldSchema.MinLength = &x
				}
			}
			if maxLen := f.Tag.Get("maxLength"); maxLen != "" {
				if v, err := strconv.Atoi(maxLen); err == nil {
					x := int64(v)
					fieldSchema.MaxLength = &x
				}
			}

			// --- NEW: Read minItems/maxItems ---
			if minItems := f.Tag.Get("minItems"); minItems != "" {
				if v, err := strconv.Atoi(minItems); err == nil {
					x := int64(v)
					fieldSchema.MinItems = &x
				}
			}
			if maxItems := f.Tag.Get("maxItems"); maxItems != "" {
				if v, err := strconv.Atoi(maxItems); err == nil {
					x := int64(v)
					fieldSchema.MaxItems = &x
				}
			}

			s.Properties[fieldName] = fieldSchema
			s.PropertyOrdering = append(s.PropertyOrdering, fieldName)

			// Required if not omitempty
			isOmitempty := false
			for _, opt := range parts[1:] {
				if opt == "omitempty" {
					isOmitempty = true
					break
				}
			}
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
