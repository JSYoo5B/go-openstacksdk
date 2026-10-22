package messages

import (
	"context"
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Resources returns a Collection bound to direct GET messages/{id} for the
// shared wait engine, such as blockstorage.WaitForDelete. Only resource.ID
// references are supported. MessageRecord keeps attributes as raw JSON, so it
// exposes no exported string attribute for status waits.
func (a *API) Resources() *resource.Collection[MessageRecord] {
	return resource.NewCollection(resource.Adapter[MessageRecord]{
		Kind: "messages",
		Get:  func(ctx context.Context, id string) (*MessageRecord, error) { return a.GetRecord(ctx, id) },
		ID:   func(v *MessageRecord) string { return rawText(v.Resource, "id") },
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
