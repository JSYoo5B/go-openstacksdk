package senlin

import (
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/gophercloudsdk/internal/jsonfilter"
)

// MatchFilters applies the Body-filter semantics of pinned Python Resource.list.
// The shared JSON comparator preserves exact numbers and strict JSON types;
// Senlin retains its established error prefix and wrapped parse causes.
func MatchFilters(body map[string]json.RawMessage, filters map[string]json.RawMessage) (bool, error) {
	matched, err := jsonfilter.MatchFilters(body, filters)
	if err != nil {
		return false, fmt.Errorf("Senlin %w", err)
	}
	return matched, nil
}
