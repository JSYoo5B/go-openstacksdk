package capabilities

import (
	"context"
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Resources returns a Collection bound to direct GET capabilities/{host} for
// the shared wait engine, such as blockstorage.WaitForDelete. The reference is
// the backend host; only resource.ID references are supported. Record keeps
// attributes as raw JSON, so it exposes no exported string attribute for
// status waits.
func (a *API) Resources() *resource.Collection[Record] {
	return resource.NewCollection(resource.Adapter[Record]{
		Kind: "capabilities",
		Get:  func(ctx context.Context, host string) (*Record, error) { return a.GetRecord(ctx, host) },
		ID:   func(v *Record) string { return rawText(v.Resource, "id") },
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
