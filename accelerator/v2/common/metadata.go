// Package common contains response metadata shared by Cyborg v2 resources.
package common

import (
	"bytes"
	"fmt"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

type Link = resource.Link

// Metadata preserves the complete object, including unknown fields and the
// difference between omitted, null and empty values. Timestamps retain the
// service's original string because Cyborg also returns non-RFC3339 formats.
type Metadata = resource.Metadata

// Decode is used by resource decoders; a successful response must be an object.
func Decode(data []byte, target any, metadata *Metadata) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return fmt.Errorf("Cyborg resource must be a JSON object")
	}
	return resource.DecodeObject(data, target, metadata)
}
