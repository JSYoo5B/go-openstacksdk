// Package troveroot reads the pinned Trove root-enabled response without the
// native extractor's unchecked assertion before its stored error.
package troveroot

import (
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
)

// EnvelopeError describes a successful response with an invalid decoded shape.
// The native result does not preserve the accepted status or response bytes.
type EnvelopeError struct {
	Actual string
}

func (e *EnvelopeError) Error() string {
	return fmt.Sprintf("Trove root-enabled response must be a JSON object (got %s)", e.Actual)
}

// Extract preserves the native comparison to literal true for every decoded
// map, including typed nil maps and absent, null, or non-boolean field values.
func Extract(result gophercloud.Result) (bool, error) {
	if result.Err != nil {
		return false, result.Err
	}
	body, ok := result.Body.(map[string]any)
	if !ok {
		return false, &EnvelopeError{Actual: fmt.Sprintf("%T", result.Body)}
	}
	return body["rootEnabled"] == true, nil
}
