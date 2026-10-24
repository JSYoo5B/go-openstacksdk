package defaultrules

import (
	"context"
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Resources returns a Collection bound to direct GET
// default-security-group-rules/{id} for the shared wait engine, such as
// network.WaitForStatus and network.WaitForDelete. Only resource.ID references
// are supported; listing, name lookup and deletion stay on API. The record
// view keeps attributes as raw JSON keyed by Python attribute name;
// resource.WithStatusAttribute selects one of them by exact name and requires
// a JSON string value.
func (a *API) Resources() *resource.Collection[Record] {
	return resource.NewCollection(resource.Adapter[Record]{
		WaitAttributes: func(v *Record) map[string]json.RawMessage { return resource.RawResourceView(v.Resource) },
		Kind:           kind,
		Get:            func(ctx context.Context, id string) (*Record, error) { return a.Get(ctx, id) },
		ID:             func(v *Record) string { return v.Text("id") },
	})
}
