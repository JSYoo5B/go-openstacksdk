package shareattachments

import (
	"bytes"
	"encoding/json"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
)

// Source Resource.list uses _dict_filter for a dictionary condition and Python
// == for its other leaves. The ordinary Collection's exact JSON policy is kept
// unchanged; Source float and Boolean rules are an explicit binding opt-in.
func sourceLocalMatch(actual, wanted json.RawMessage) (bool, error) {
	if actual == nil {
		actual = json.RawMessage("null")
	}
	if wanted == nil {
		wanted = json.RawMessage("null")
	}
	if raw := bytes.TrimSpace(wanted); len(raw) > 0 && raw[0] == '{' {
		members, err := cloudfilter.ObjectMembers(wanted)
		if err != nil {
			return false, err
		}
		truthy, err := cloudfilter.PythonJSONTruthy(actual)
		if err != nil || !truthy {
			return false, err
		}
		// Source never accesses .get when its wanted dictionary is empty.
		if len(members) == 0 {
			return true, nil
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(actual, &fields); err != nil || fields == nil {
			return false, invalid("local dictionary filter requires an actual mapping")
		}
		for _, member := range members {
			matches, err := sourceLocalMatch(fields[member.Key], member.Value)
			if err != nil || !matches {
				return matches, err
			}
		}
		return true, nil
	}
	return jsonfilter.EqualPythonLoadedJSON(actual, wanted)
}
func sourceLocalFilters(value *Record, filters map[string]json.RawMessage) (bool, error) {
	keys := make([]string, 0, len(filters))
	for key := range filters {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		matches, err := sourceLocalMatch(value.Resource.Body[key], filters[key])
		if err != nil || !matches {
			return matches, err
		}
	}
	return true, nil
}
