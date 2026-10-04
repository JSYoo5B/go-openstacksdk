package jmespath

import (
	"bytes"
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

func decodeJSON(data []byte) (any, error) {
	if !utf8.Valid(data) || !json.Valid(data) {
		return nil, fmt.Errorf("literal must be valid UTF-8 JSON")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

func cloneLiteral(value any) any {
	switch value := value.(type) {
	case []any:
		result := make([]any, len(value))
		for i, item := range value {
			result[i] = cloneLiteral(item)
		}
		return result
	case map[string]any:
		result := make(map[string]any, len(value))
		for key, item := range value {
			result[key] = cloneLiteral(item)
		}
		return result
	default:
		return value
	}
}
