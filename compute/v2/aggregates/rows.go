package aggregates

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"iter"

	"github.com/JSYoo5B/go-openstacksdk/resource"

	upstream "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/aggregates"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

// AggregateRow is one GET /os-aggregates row: the typed Aggregate together
// with the JSON object Nova returned, so null values and the original
// timestamp strings stay visible.
type AggregateRow struct {
	Aggregate
	Raw json.RawMessage
}

// ListRows lists aggregates like List but keeps each row's JSON object. Number
// values in Raw are re-encoded from the decoded page body.
func (a *API) ListRows(ctx context.Context) iter.Seq2[*AggregateRow, error] {
	return resource.StreamWithControl(ctx, upstream.List(a.client), extractRows, resource.ListControl{})
}

func extractRows(page pagination.Page) ([]AggregateRow, error) {
	typed, err := upstream.ExtractAggregates(page)
	if err != nil {
		return nil, err
	}
	var wire struct {
		Aggregates []json.RawMessage `json:"aggregates"`
	}
	if err := page.(upstream.AggregatesPage).ExtractInto(&wire); err != nil {
		return nil, err
	}
	if len(wire.Aggregates) != len(typed) {
		return nil, fmt.Errorf("aggregate list rows changed while decoding")
	}
	rows := make([]AggregateRow, len(typed))
	for i := range typed {
		raw := bytes.TrimSpace(wire.Aggregates[i])
		if len(raw) == 0 || raw[0] != '{' {
			return nil, fmt.Errorf("aggregate list row %d is not a JSON object", i)
		}
		rows[i] = AggregateRow{Aggregate: typed[i], Raw: bytes.Clone(raw)}
	}
	return rows, nil
}
