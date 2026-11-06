package neutronfind

import (
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/request"
)

// KeywordJSON returns the Source keyword value one request.Query call
// carries: the JSON marshaled when the call was constructed and its marshal
// error. The Proxy argument jmespath_filters is evaluated only as a Python
// string, so a value that does not marshal to a JSON string becomes null,
// which Source ignores, and its marshal error is discarded.
func KeywordJSON(entry request.CallEntry) (json.RawMessage, error) {
	if entry.Key != "jmespath_filters" || entry.Form == request.QueryText {
		return entry.JSON, entry.JSONErr
	}
	if entry.JSONErr == nil && len(entry.JSON) > 0 && entry.JSON[0] == '"' {
		return entry.JSON, nil
	}
	return json.RawMessage("null"), nil
}
