package cloudsnapshot

import (
	"bytes"
	"encoding/json"
	"sort"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudfilter"
)

// ValidateCreateInput checks the literal body without selecting a service.
// It deliberately leaves reached wait/route validation to execution.
func ValidateCreateInput(volumeID string, policy CreateOptions) error {
	_, _, err := compileCreateBody(volumeID, cloneCreateOptions(policy))
	return err
}

func compileCreateBody(volumeID string, policy CreateOptions) (json.RawMessage, mutationState, error) {
	var state mutationState
	if !utf8.ValidString(volumeID) {
		return nil, state, invalid("snapshot volume_id must be UTF-8")
	}
	fields := cloneMutationFields(policy.Attributes.Fields)
	if fields == nil {
		fields = make(map[string]json.RawMessage)
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		switch key {
		case "name", "display_name", "description", "display_description":
		default:
			return nil, state, invalid("snapshot cloud creation has no %q attribute", key)
		}
	}
	typed := map[string]*string{"name": policy.Attributes.Name, "display_name": policy.Attributes.DisplayName,
		"description": policy.Attributes.Description, "display_description": policy.Attributes.DisplayDescription}
	for key, value := range typed {
		if value != nil {
			fields[key], _ = json.Marshal(*value)
		}
	}
	force := false
	if policy.Force != nil {
		force = *policy.Force
	}
	volume, _ := json.Marshal(volumeID)
	forceJSON, _ := json.Marshal(force)
	body := map[string]json.RawMessage{"volume_id": volume, "force": forceJSON}
	for _, aliases := range [][2]string{{"name", "display_name"}, {"description", "display_description"}} {
		key := aliases[0]
		raw, present := fields[key]
		if !present {
			key = aliases[1]
			raw, present = fields[key]
		}
		if !present {
			continue
		}
		if text := typed[key]; text != nil && !utf8.ValidString(*text) {
			return nil, state, invalid("snapshot %s must be UTF-8", key)
		}
		if raw == nil {
			raw = json.RawMessage("null")
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, state, invalid("snapshot %s must contain UTF-8 JSON", key)
		}
		truthy, err := cloudfilter.PythonTruthy(raw)
		if err != nil {
			return nil, state, invalid("snapshot %s: %v", key, err)
		}
		if truthy {
			body[aliases[0]] = bytes.Clone(raw)
		}
	}
	seed, err := json.Marshal(body)
	if err != nil {
		return nil, state, invalid("snapshot creation body: %v", err)
	}
	if err := state.overlay(seed); err != nil {
		return nil, state, invalid("snapshot creation state: %v", err)
	}
	envelope, err := json.Marshal(map[string]json.RawMessage{"snapshot": seed})
	return envelope, state, err
}
