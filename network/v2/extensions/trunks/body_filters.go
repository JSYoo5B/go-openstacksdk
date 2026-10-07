package trunks

import (
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// trunkBodyFilterValue keeps the two non-query Body attributes separate from
// server query conditions and the native model's zero-value projections.
func trunkBodyFilterValue(record *resource.BodyRecord[Trunk], key string) (json.RawMessage, error) {
	if record == nil {
		return nil, fmt.Errorf("%w: trunk Body record is required", resource.ErrInvalidOption)
	}
	if record.Fields == nil {
		return nil, fmt.Errorf("%w: null trunk list row is not an object", resource.ErrInvalidOption)
	}
	switch key {
	case "id", "tenant_id":
		return resource.BodyRecordField(record.Fields, key, resource.BodyFieldJSON)
	default:
		return nil, fmt.Errorf("%w: unsupported trunk Body filter %q", resource.ErrInvalidOption, key)
	}
}
