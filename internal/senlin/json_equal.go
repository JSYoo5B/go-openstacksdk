package senlin

import (
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
)

// EqualJSON compares complete valid JSON values without losing number precision.
// Object order and decimal spelling do not affect equality; absent, invalid,
// boolean and numeric values remain distinct.
func EqualJSON(left, right json.RawMessage) bool {
	equal, err := jsonfilter.EqualJSON(left, right)
	return err == nil && equal
}
