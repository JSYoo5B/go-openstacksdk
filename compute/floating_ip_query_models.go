package compute

import (
	"encoding/json"
	"fmt"

	"gophercloudsdk/resource"
)

// FloatingIPRecord separates the returned cloud resource from the original
// service row. Normalized Nova views can contain source-synthetic status;
// Wire always retains actual fields, including unknown and nullable values.
type FloatingIPRecord struct {
	Backend             FloatingIPSource
	NormalizationSource FloatingIPSource
	Normalized          bool
	Resource            *resource.RawResource
	Wire                *resource.RawResource
}

// FloatingIPQueryResponse preserves one physical page/member response.
type FloatingIPQueryResponse struct {
	resource.Metadata
	Backend  FloatingIPSource
	Envelope json.RawMessage
}

// Value may be arbitrary expression output. FloatingIPs identifies source rows
// only for ordinary selection; Pages survive errors without asserting a list.
type FloatingIPQueryResult struct {
	Backend            FloatingIPSource
	Value              json.RawMessage
	FloatingIPs        []*FloatingIPRecord
	Pages              []*FloatingIPQueryResponse
	FallbackError      error
	SuppressedNotFound error
}
type GetFloatingIPResult struct {
	Backend            FloatingIPSource
	Value              json.RawMessage
	FloatingIP         *FloatingIPRecord
	Observed           *FloatingIPQueryResponse
	Pages              []*FloatingIPQueryResponse
	FallbackError      error
	SuppressedNotFound error
}
type FloatingIPPoolQueryResult struct {
	Value json.RawMessage
	Pools []*resource.RawResource
	Pages []*FloatingIPQueryResponse
}
type FloatingIPSelectionError struct {
	ID     string
	Length int
}

func (e *FloatingIPSelectionError) Error() string {
	return fmt.Sprintf("floating IP %q has multiple matches (length %d)", e.ID, e.Length)
}
func (e *FloatingIPSelectionError) Unwrap() error { return resource.ErrAmbiguous }
