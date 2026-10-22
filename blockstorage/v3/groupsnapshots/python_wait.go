package groupsnapshots

import (
	"context"
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Resources returns a Collection bound to direct GET group_snapshots/{id} for the
// shared wait engine, such as blockstorage.WaitForState and
// blockstorage.WaitForDelete. Only resource.ID references are supported;
// listing, lookup and deletion stay on API. Status reads the declared status
// attribute from the record view (empty when absent, null or not a string),
// so status waits and the "deleted" deletion check use it. The record keeps
// other attributes as raw JSON, so resource.WithStatusAttribute cannot
// select them.
func (a *API) Resources() *resource.Collection[GroupSnapshotRecord] {
	return resource.NewCollection(resource.Adapter[GroupSnapshotRecord]{
		Kind:   "group_snapshots",
		Get:    func(ctx context.Context, id string) (*GroupSnapshotRecord, error) { return a.GetRecord(ctx, id) },
		ID:     func(v *GroupSnapshotRecord) string { return waitText(v.Resource, "id") },
		Status: func(v *GroupSnapshotRecord) string { return waitText(v.Resource, "status") },
	})
}

func waitText(value *resource.RawResource, key string) string {
	if value == nil {
		return ""
	}
	var text string
	_ = json.Unmarshal(value.Body[key], &text)
	return text
}
