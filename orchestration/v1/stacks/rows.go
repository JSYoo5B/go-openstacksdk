package stacks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2/pagination"
)

// StackRow is one stack list row: the typed summary (Detailed=false) together
// with the exact JSON object Heat returned, including fields that the native
// ListedStack model does not declare.
type StackRow struct {
	StackResource
	Raw json.RawMessage
}

// AllRows collects every page of GET /stacks like All, but keeps each row's
// original JSON object. Paging, marker fallback and cycle guards are shared
// with All; list options have the same meaning.
func (s *StackResources) AllRows(ctx context.Context, options ...resource.ListOption) ([]*StackRow, error) {
	binding := resource.Adapter[StackRow]{
		Kind:       "stacks",
		ValidateID: validateSegment,
		ID:         func(value *StackRow) string { return value.ID },
		Name:       func(value *StackRow) string { return value.Name },
		List:       func(query url.Values) pagination.Pager { return resourcePager(s.api.client, query) },
		Extract:    extractRows,
		Status:     func(value *StackRow) string { return value.Status },
		Failed:     failedStatus,
	}
	return resource.NewCollection(binding).All(ctx, options...)
}

func extractRows(page pagination.Page) ([]StackRow, error) {
	typed, err := extractResources(page)
	if err != nil {
		return nil, err
	}
	var wire struct {
		Stacks []json.RawMessage `json:"stacks"`
	}
	if err := page.(resourcePage).ExtractInto(&wire); err != nil {
		return nil, err
	}
	if len(wire.Stacks) != len(typed) {
		return nil, fmt.Errorf("stack list rows changed while decoding")
	}
	rows := make([]StackRow, len(typed))
	for i := range typed {
		raw := bytes.TrimSpace(wire.Stacks[i])
		if len(raw) == 0 || raw[0] != '{' {
			return nil, fmt.Errorf("stack list row %d is not a JSON object", i)
		}
		rows[i] = StackRow{StackResource: typed[i], Raw: bytes.Clone(raw)}
	}
	return rows, nil
}
