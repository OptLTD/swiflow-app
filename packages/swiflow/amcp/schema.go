package amcp

import (
	"encoding/json"

	"github.com/google/jsonschema-go/jsonschema"
)

// ... existing code ...
// MapToSchema 将 map[string]any 转为 *jsonschema.Schema
func MapToSchema(m map[string]any) (*jsonschema.Schema, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	schema := new(jsonschema.Schema)
	if err := schema.UnmarshalJSON(b); err != nil {
		return nil, err
	}
	return schema, nil
}

func ToJsonSchema(s string) (*jsonschema.Schema, error) {
	schema := new(jsonschema.Schema)
	if err := schema.UnmarshalJSON([]byte(s)); err != nil {
		return nil, err
	}
	return schema, nil
}
