package templateloader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"gopkg.in/yaml.v2"
)

// yamlNode decodes YAML into JSON-compatible values the way Heat's loader does:
// mappings become map[string]any, sequences []any, timestamps keep their
// original text (never time.Time) and other scalars keep their resolved type.
type yamlNode struct{ value any }

func (n *yamlNode) UnmarshalYAML(unmarshal func(any) error) error {
	var generic any
	if err := unmarshal(&generic); err != nil {
		return err
	}
	switch generic.(type) {
	case map[any]any:
		var m map[string]yamlNode
		if err := unmarshal(&m); err != nil {
			return err
		}
		out := make(map[string]any, len(m))
		for key, value := range m {
			out[key] = value.value
		}
		n.value = out
	case []any:
		var s []yamlNode
		if err := unmarshal(&s); err != nil {
			return err
		}
		out := make([]any, len(s))
		for i, value := range s {
			out[i] = value.value
		}
		n.value = out
	case time.Time:
		var text string
		if err := unmarshal(&text); err != nil {
			return err
		}
		n.value = text
	default:
		n.value = generic
	}
	return nil
}

func parseYAML(data []byte) (any, error) {
	var root yamlNode
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	return root.value, nil
}

// parseTemplate decodes a JSON (leading '{') or YAML Heat template. The result
// must be a mapping that declares a template format version.
func parseTemplate(text string) (map[string]any, error) {
	text = strings.TrimSpace(text)
	var value any
	if strings.HasPrefix(text, "{") {
		decoder := json.NewDecoder(strings.NewReader(text))
		decoder.UseNumber()
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		if decoder.More() {
			return nil, fmt.Errorf("extra data after JSON template")
		}
	} else {
		var err error
		if value, err = parseYAML([]byte(text)); err != nil {
			return nil, err
		}
	}
	template, _ := value.(map[string]any)
	for _, key := range []string{"HeatTemplateFormatVersion", "heat_template_version", "AWSTemplateFormatVersion"} {
		if _, ok := template[key]; ok {
			return template, nil
		}
	}
	return nil, fmt.Errorf("Template format version not found.")
}

var environmentSections = map[string]bool{
	"parameter_defaults": true, "parameters": true, "resource_registry": true,
	"encrypted_param_names": true, "event_sinks": true, "parameter_merge_strategies": true,
}

// parseEnvironment decodes a YAML (or JSON) environment. Empty input is an
// empty environment; unknown top-level sections are rejected.
func parseEnvironment(data []byte) (map[string]any, error) {
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("environment is not valid UTF-8")
	}
	value, err := parseYAML(data)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return map[string]any{}, nil
	}
	env, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("The environment is not a valid YAML mapping data type.")
	}
	for key := range env {
		if !environmentSections[key] {
			return nil, fmt.Errorf("environment has wrong section %q", key)
		}
	}
	return env, nil
}

// isTemplate reports whether a file's content parses as a Heat template.
func isTemplate(content string) bool {
	_, err := parseTemplate(content)
	return err == nil
}

// dumpJSON serializes a nested template for the files map. The output is
// compact JSON with sorted keys and unescaped HTML characters.
func dumpJSON(value any) (string, error) {
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return "", err
	}
	return strings.TrimSuffix(out.String(), "\n"), nil
}

// deepUpdate merges mappings recursively into old; other values replace.
func deepUpdate(old map[string]any, update map[string]any) {
	for key, value := range update {
		if nested, ok := value.(map[string]any); ok {
			existing, _ := old[key].(map[string]any)
			if existing == nil {
				existing = map[string]any{}
			}
			deepUpdate(existing, nested)
			old[key] = existing
			continue
		}
		old[key] = value
	}
}
