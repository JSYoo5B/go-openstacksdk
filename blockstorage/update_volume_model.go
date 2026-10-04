package blockstorage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"gophercloudsdk/internal/jsonfilter"
	"gophercloudsdk/resource"
)

// Canonicalize known raw attributes in actual JSON member order. This retains
// lookup alias semantics without comparing the coerced to_dict descriptor view.
// Unknown original fields remain a raw Go extension; unknown response fields
// are excluded by the response merge below, as in the source Resource.
func volumeMutationFields(raw json.RawMessage) (map[string]json.RawMessage, error) {
	if !utf8.Valid(raw) || !json.Valid(raw) || len(bytes.TrimSpace(raw)) == 0 || bytes.TrimSpace(raw)[0] != '{' {
		return nil, fmt.Errorf("volume mutation fields must be a UTF-8 JSON object")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	_, _ = decoder.Token()
	fields := make(map[string]json.RawMessage)
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		name := key.(string)
		for _, descriptor := range volumeSearchDescriptors {
			if name == descriptor.attribute || name == descriptor.wire {
				name = descriptor.wire
				break
			}
		}
		fields[name] = bytes.Clone(value)
	}
	return fields, nil
}

func volumeUpdateProposal(raw json.RawMessage, requested map[string]json.RawMessage) (map[string]json.RawMessage, map[string]json.RawMessage, error) {
	fields, err := volumeMutationFields(raw)
	if err != nil {
		return nil, nil, err
	}
	dirty := make(map[string]json.RawMessage)
	for key, value := range requested {
		if prior, exists := fields[key]; exists {
			equal, err := jsonfilter.EqualPythonJSON(prior, value)
			if err != nil {
				return nil, nil, err
			}
			if equal {
				continue
			} // Equal assignment does not replace source raw state.
		}
		fields[key] = bytes.Clone(value)
		dirty[key] = bytes.Clone(value)
	}
	return fields, dirty, nil
}

func volumeMutationView(fields map[string]json.RawMessage, location resource.CloudLocation) (json.RawMessage, error) {
	raw, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}
	base, err := volumeSearchView(raw, nil)
	if err != nil {
		return nil, err
	}
	var normalized map[string]json.RawMessage
	if err := json.Unmarshal(base, &normalized); err != nil {
		return nil, err
	}
	computed, err := location.ForResource(normalized["project_id"], normalized["availability_zone"])
	if err != nil {
		return nil, err
	}
	return volumeSearchView(raw, computed)
}

// Resource._translate_response tolerates JSON decoder ValueError (including
// accepted empty/malformed bodies), but valid nonobjects and null envelopes
// cannot be consumed. Partial/flat objects overlay only recognized fields.
func mergeVolumeUpdateResponse(fields map[string]json.RawMessage, body []byte) error {
	if !utf8.Valid(body) {
		return fmt.Errorf("volume update response must be valid UTF-8")
	}
	if !json.Valid(body) {
		return nil
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil || envelope == nil {
		return fmt.Errorf("volume update response must be a nonnull JSON object")
	}
	raw := json.RawMessage(body)
	if selected, present := envelope["volume"]; present {
		raw = selected
	}
	server, err := volumeMutationFields(raw)
	if err != nil {
		return err
	}
	for _, descriptor := range volumeSearchDescriptors {
		if value, exists := server[descriptor.wire]; exists {
			fields[descriptor.wire] = bytes.Clone(value)
		}
	}
	return nil
}
