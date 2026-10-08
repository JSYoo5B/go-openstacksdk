package cloudread

import (
	"bytes"
	"encoding/json"
	"maps"
	"net/url"
	"slices"

	"github.com/JSYoo5B/gophercloudsdk/request"
)

// OwnReadConfig snapshots the mutable request carriers around an option
// callback. Bindings still own their typed options and argument payloads;
// arbitrary application values cannot be deep-copied by the common reader.
func OwnReadConfig[T any](config *request.Config[T]) {
	query := make(url.Values, len(config.Query))
	for key, values := range config.Query {
		query[key] = slices.Clone(values)
	}
	config.Query = query
	config.Headers = maps.Clone(config.Headers)
	if config.Headers == nil {
		config.Headers = make(map[string]string)
	}
	fields := make(map[string]json.RawMessage, len(config.Fields))
	for key, raw := range config.Fields {
		fields[key] = bytes.Clone(raw)
	}
	config.Fields = fields
	config.Arguments = maps.Clone(config.Arguments)
	if config.Arguments == nil {
		config.Arguments = make(map[string]any)
	}
}
