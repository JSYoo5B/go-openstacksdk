package cloudlimits

import (
	"bytes"
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func observed(response *rest.Response) *Page {
	if response == nil {
		return nil
	}
	return &Page{Body: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
}

// Source Resource.fetch accepts flat or enveloped objects and tolerates JSON
// ValueError. A valid wrong shape and descriptor failures remain physical errors.
func responseObject(response *rest.Response, key string, tolerate bool) (json.RawMessage, error) {
	if !utf8.Valid(response.Body) {
		return nil, response.Fail(fmt.Errorf("response must be UTF-8 JSON"))
	}
	if !json.Valid(response.Body) && tolerate {
		return json.RawMessage("{}"), nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(response.Body, &fields); err != nil {
		return nil, response.Fail(err)
	}
	if fields == nil {
		return nil, response.Fail(fmt.Errorf("response must be a nonnull JSON object"))
	}
	raw, exists := fields[key]
	if !exists {
		raw = response.Body
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return nil, response.Fail(err)
	}
	if object == nil {
		return nil, response.Fail(fmt.Errorf("response %q must be a nonnull JSON object", key))
	}
	return bytes.Clone(raw), nil
}

func rawResource(raw json.RawMessage, response *rest.Response) (*resource.RawResource, error) {
	var value resource.RawResource
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, response.Fail(err)
	}
	value.Header = response.Header.Clone()
	value.StatusCode = response.StatusCode
	return &value, nil
}

func sourceCodes() []int {
	codes := make([]int, 300)
	for i := range codes {
		codes[i] = 100 + i
	}
	return codes
}
