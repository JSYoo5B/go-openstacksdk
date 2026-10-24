package messages

import (
	"context"
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Resources returns a Collection bound to direct GET messages/{id} for the
// shared wait engine, such as blockstorage.WaitForDelete. Only resource.ID
// references are supported. The record view keeps attributes as raw JSON keyed
// by Python attribute name; resource.WithStatusAttribute selects one of them
// by exact name and requires a JSON string value.
func (a *API) Resources() *resource.Collection[MessageRecord] {
	return resource.NewCollection(resource.Adapter[MessageRecord]{
		WaitAttributes: func(v *MessageRecord) map[string]json.RawMessage { return resource.RawResourceView(v.Resource) },
		Kind:           "messages",
		Get:            func(ctx context.Context, id string) (*MessageRecord, error) { return a.GetRecord(ctx, id) },
		ID:             func(v *MessageRecord) string { return rawText(v.Resource, "id") },
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
