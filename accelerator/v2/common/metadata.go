// Package common contains response metadata shared by Cyborg v2 resources.
package common

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

// Metadata preserves the complete object, including unknown fields and the
// difference between omitted, null and empty values. Timestamps retain the
// service's original string because Cyborg also returns non-RFC3339 formats.
type Metadata struct {
	CreatedAt  *string                    `json:"created_at"`
	UpdatedAt  *string                    `json:"updated_at"`
	Links      []Link                     `json:"links"`
	Body       map[string]json.RawMessage `json:"-"`
	Header     http.Header                `json:"-"`
	StatusCode int                        `json:"-"`
}

// Decode is used by resource decoders; a successful response must be an object.
func Decode(data []byte, target any, metadata *Metadata) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return fmt.Errorf("Cyborg resource must be a JSON object")
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
