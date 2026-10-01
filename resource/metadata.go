package resource

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Link struct {
	Href string `json:"href"`
	Rel  string `json:"rel"`
}

// Metadata retains a resource's original fields and HTTP evidence. Timestamp
// strings preserve service formats; Body distinguishes omission from JSON null
// and keeps extension numbers without converting them through float64.
type Metadata struct {
	CreatedAt  *string                    `json:"created_at"`
	UpdatedAt  *string                    `json:"updated_at"`
	Links      []Link                     `json:"links"`
	Body       map[string]json.RawMessage `json:"-"`
	Header     http.Header                `json:"-"`
	StatusCode int                        `json:"-"`
}

// DecodeObject is used by SDK-owned model decoders with an unmarshal-free alias
// of their model as target. It requires an object and retains its exact fields.
func DecodeObject(data []byte, target any, metadata *Metadata) error {
	if metadata == nil {
		return fmt.Errorf("%w: response metadata is required", ErrInvalidOption)
	}
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return fmt.Errorf("resource must be a JSON object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return err
	}
	metadata.Body = fields
	return nil
}

// ResponseError preserves an accepted response when reading or decoding fails.
// A mutation may already have succeeded; this error does not trigger a resend.
type ResponseError struct {
	Body       []byte
	Header     http.Header
	StatusCode int
	Cause      error
}

func (e *ResponseError) Error() string {
	return fmt.Sprintf("response HTTP %d: %v", e.StatusCode, e.Cause)
}

func (e *ResponseError) Unwrap() error { return e.Cause }
