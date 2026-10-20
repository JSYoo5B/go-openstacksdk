package cindermetadata

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/metadata"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

func decode(response *rest.Response) (*metadata.Result, error) {
	object, err := response.Object("metadata")
	if err != nil {
		return nil, err
	}
	if !utf8.Valid(object) {
		return nil, response.Fail(fmt.Errorf("metadata object must contain valid UTF-8 JSON text"))
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(object, &fields); err != nil {
		return nil, response.Fail(err)
	}
	values := make(map[string]string, len(fields))
	for key, raw := range fields {
		var value any
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, response.Fail(err)
		}
		text, ok := value.(string)
		if !ok {
			return nil, response.Fail(fmt.Errorf("metadata value %q must be a JSON string", key))
		}
		values[key] = text
	}
	return &metadata.Result{Metadata: values, Body: append(json.RawMessage(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}, nil
}
