package quotasets

import (
	"context"
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// QuotaClassResources returns a Collection bound to direct GET
// os-quota-class-sets/{name} for the shared wait engine, such as
// blockstorage.WaitForDelete. The reference is the quota class name; only
// resource.ID references are supported. The record view keeps attributes as
// raw JSON keyed by Python attribute name; resource.WithStatusAttribute
// selects one of them by exact name and requires a JSON string value.
func (a *API) QuotaClassResources() *resource.Collection[QuotaClassRecord] {
	return resource.NewCollection(resource.Adapter[QuotaClassRecord]{
		WaitAttributes: func(v *QuotaClassRecord) map[string]json.RawMessage { return resource.RawResourceView(v.Resource) },
		Kind:           "quota_class_sets",
		Get: func(ctx context.Context, name string) (*QuotaClassRecord, error) {
			return a.GetQuotaClassRecord(ctx, WithQuotaClassName(name))
		},
		// The class body carries its name as id; Name references are not
		// supported, so ID is informational only.
		ID: func(*QuotaClassRecord) string { return "" },
	})
}
