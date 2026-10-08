package blockstorage

import (
	"bytes"
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/internal/cindervolume"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Canonicalize known raw attributes in actual JSON member order. This retains
// lookup alias semantics without comparing the coerced to_dict descriptor view.
// Unknown original fields remain a raw Go extension; unknown response fields
// are excluded by the response merge below, as in the source Resource.
func volumeMutationFields(raw json.RawMessage) (map[string]json.RawMessage, error) {
	return cindervolume.Fields(raw)
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
	return cindervolume.MergeResponse(fields, body)
}
