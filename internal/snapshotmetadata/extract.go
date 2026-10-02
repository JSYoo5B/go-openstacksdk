// Package snapshotmetadata safely reads the pinned Cinder snapshot metadata
// response without using the unrelated promoted Snapshot extractor.
package snapshotmetadata

import (
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
)

// EnvelopeError describes an invalid decoded response shape. The native result
// does not retain the original response bytes or accepted status code.
type EnvelopeError struct {
	Field  string
	Actual string
}

func (e *EnvelopeError) Error() string {
	return fmt.Sprintf("snapshot metadata response: %s must be a non-null JSON object (got %s)", e.Field, e.Actual)
}

// Extract returns the actual decoded metadata object. The native transport has
// already decoded numbers with UseNumber; do not marshal or decode it again.
func Extract(result gophercloud.Result) (map[string]any, error) {
	if result.Err != nil {
		return nil, result.Err
	}
	body, ok := result.Body.(map[string]any)
	if !ok || body == nil {
		return nil, &EnvelopeError{Field: "response", Actual: fmt.Sprintf("%T", result.Body)}
	}
	value, present := body["metadata"]
	if !present {
		return nil, &EnvelopeError{Field: "metadata", Actual: "absent"}
	}
	metadata, ok := value.(map[string]any)
	if !ok || metadata == nil {
		return nil, &EnvelopeError{Field: "metadata", Actual: fmt.Sprintf("%T", value)}
	}
	return metadata, nil
}
