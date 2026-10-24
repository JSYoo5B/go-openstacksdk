package domains

import (
	"context"
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ConfigResources returns a Collection bound to direct GET
// domains/{domain_id}/config, keyed by domain ID, for the shared wait engine,
// such as identity.WaitForDelete. Keystone answers 404 when the domain has no
// configuration. Only resource.ID references are supported. The record view
// keeps attributes as raw JSON keyed by Python attribute name;
// resource.WithStatusAttribute selects one of them by exact name and requires
// a JSON string value.
func (a *API) ConfigResources() *resource.Collection[DomainConfigRecord] {
	return resource.NewCollection(resource.Adapter[DomainConfigRecord]{
		WaitAttributes: func(v *DomainConfigRecord) map[string]json.RawMessage { return resource.RawResourceView(v.Resource) },
		Kind:           "domainconfigs",
		Get: func(ctx context.Context, domainID string) (*DomainConfigRecord, error) {
			return a.GetConfigRecord(ctx, domainID)
		},
		// The configuration body carries no domain ID. ID is used only for
		// Name references, which this collection does not support.
		ID: func(*DomainConfigRecord) string { return "" },
	})
}
