package routers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// RouterJSONResult owns the whole response.json() value for interface and
// single-gateway operations. Value includes any returned envelope and may be
// an object, list, string, number, bool or explicit null. Wire exists only for
// whole JSON objects; no Router descriptor defaults or cache seed are applied.
// Envelope/Header/StatusCode preserve actual HTTP evidence independently.
// Physical read/Close or decode failures may return observation without Value.
type RouterJSONResult struct {
	Value      json.RawMessage
	Wire       *resource.RawResource
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
}

func routerJSONObservation(response *rest.Response) *RouterJSONResult {
	if response == nil {
		return nil
	}
	value := &RouterJSONResult{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	var wire resource.RawResource
	if json.Unmarshal(response.Body, &wire) == nil {
		wire.Header, wire.StatusCode = response.Header.Clone(), response.StatusCode
		value.Wire = wire.Clone()
	}
	return value
}
func routerJSONFromResponse(response *rest.Response) (*RouterJSONResult, error) {
	value := routerJSONObservation(response)
	if value == nil {
		return nil, routerActionInvalid("router JSON response is required")
	}
	if !utf8.Valid(response.Body) {
		return value, routerActionInvalid("router action response must be UTF-8 JSON")
	}
	if len(bytes.TrimSpace(response.Body)) == 0 {
		return value, io.EOF
	}
	var raw json.RawMessage
	if err := json.Unmarshal(response.Body, &raw); err != nil {
		return value, err
	}
	value.Value = bytes.Clone(raw)
	return value, nil
}
