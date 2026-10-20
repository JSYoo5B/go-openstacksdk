package image

import (
	"encoding/json"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ImagePatch is one concrete Glance v2.1 JSON Patch operation. Add and replace
// require a JSON Value (including null); remove requires a strictly nil Value.
type ImagePatch struct {
	Op    string          `json:"op"`
	Path  string          `json:"path"`
	Value json.RawMessage `json:"value,omitempty"`
}

// Clone without collapsing a supplied empty RawMessage into nil: remove uses
// that presence distinction, and add/replace reject either empty representation.
func copyImagePatchRaw(value json.RawMessage) json.RawMessage {
	if value == nil {
		return nil
	}
	owned := make(json.RawMessage, len(value))
	copy(owned, value)
	return owned
}
func copyImagePatch(value ImagePatch) ImagePatch {
	value.Value = copyImagePatchRaw(value.Value)
	return value
}
func copyImagePatches(value []ImagePatch) []ImagePatch {
	if value == nil {
		return nil
	}
	owned := make([]ImagePatch, len(value))
	for i, change := range value {
		owned[i] = copyImagePatch(change)
	}
	return owned
}
func copyImagePatchMap(value map[string]json.RawMessage) map[string]json.RawMessage {
	if value == nil {
		return nil
	}
	owned := make(map[string]json.RawMessage, len(value))
	for key, raw := range value {
		owned[key] = copyImagePatchRaw(raw)
	}
	return owned
}
func pythonStripSpace(value rune) bool {
	return unicode.IsSpace(value) || value >= '\u001c' && value <= '\u001f'
}
func validateImagePatchField(value string) error {
	if value == "" || !utf8.ValidString(value) {
		return uploadInvalid("image field must be nonempty valid UTF-8")
	}
	first, _ := utf8.DecodeRuneInString(value)
	last, _ := utf8.DecodeLastRuneInString(value)
	if pythonStripSpace(first) || pythonStripSpace(last) {
		return uploadInvalid("image field must retain its identity after server whitespace stripping")
	}
	return nil
}
func imagePatchFieldPath(value string) string {
	return "/" + strings.ReplaceAll(strings.ReplaceAll(value, "~", "~0"), "/", "~1")
}
func validateImagePatchPath(value string) error {
	if !utf8.ValidString(value) || !strings.HasPrefix(value, "/") {
		return uploadInvalid("image patch path must be a valid UTF-8 JSON pointer")
	}
	for _, token := range strings.Split(value[1:], "/") {
		for i := 0; i < len(token); i++ {
			if token[i] == '~' {
				if i+1 == len(token) || token[i+1] != '0' && token[i+1] != '1' {
					return uploadInvalid("image patch pointer has an invalid escape")
				}
				i++
			}
		}
		decoded := strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")
		if err := validateImagePatchField(decoded); err != nil {
			return err
		}
	}
	return nil
}
func validateImagePatch(value ImagePatch) error {
	if value.Op != "add" && value.Op != "replace" && value.Op != "remove" {
		return uploadInvalid("image patch operation must be add, replace or remove")
	}
	if err := validateImagePatchPath(value.Path); err != nil {
		return err
	}
	if value.Op == "remove" {
		if value.Value != nil {
			return uploadInvalid("image remove patch must omit its value")
		}
	} else if value.Value == nil || !utf8.Valid(value.Value) || !json.Valid(value.Value) {
		return uploadInvalid("image add and replace require a valid UTF-8 JSON value")
	}
	return nil
}
func imagePatchFields(value map[string]json.RawMessage) ([]ImagePatch, error) {
	keys := make([]string, 0, len(value))
	for key := range value {
		if err := validateImagePatchField(key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	changes := make([]ImagePatch, 0, len(keys))
	for _, key := range keys {
		change := ImagePatch{Op: "add", Path: imagePatchFieldPath(key), Value: copyImagePatchRaw(value[key])}
		if err := validateImagePatch(change); err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}
	return changes, nil
}
func marshalImagePatchField(key string, value any) (json.RawMessage, error) {
	if err := validateImagePatchField(key); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if !utf8.Valid(encoded) || !json.Valid(encoded) {
		return nil, uploadInvalid("image field requires a valid UTF-8 JSON value")
	}
	return encoded, nil
}
