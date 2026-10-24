package capabilities

import (
	"context"
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Resources returns a Collection bound to direct GET capabilities/{host} for
// the shared wait engine, such as blockstorage.WaitForDelete. The reference is
// the backend host; only resource.ID references are supported. The record view
// keeps attributes as raw JSON keyed by Python attribute name;
// resource.WithStatusAttribute selects one of them by exact name and requires
// a JSON string value.
func (a *API) Resources() *resource.Collection[Record] {
	return resource.NewCollection(resource.Adapter[Record]{
		WaitAttributes: func(v *Record) map[string]json.RawMessage { return resource.RawResourceView(v.Resource) },
		Kind:           "capabilities",
		Get:            func(ctx context.Context, host string) (*Record, error) { return a.GetRecord(ctx, host) },
		ID:             func(v *Record) string { return rawText(v.Resource, "id") },
	})
}

func rawText(value *resource.RawResource, key string) string {
	if value == nil {
		return ""
	}
	var text string
	_ = json.Unmarshal(value.Body[key], &text)
	return text
}
