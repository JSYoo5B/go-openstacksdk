package domains

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ConfigResources returns a Collection bound to direct GET
// domains/{domain_id}/config, keyed by domain ID, for the shared wait engine,
// such as identity.WaitForDelete. Keystone answers 404 when the domain has no
// configuration. Only resource.ID references are supported. The record keeps
// option groups as raw JSON, so it exposes no exported string attribute for
// status waits.
func (a *API) ConfigResources() *resource.Collection[DomainConfigRecord] {
	return resource.NewCollection(resource.Adapter[DomainConfigRecord]{
		Kind: "domainconfigs",
		Get: func(ctx context.Context, domainID string) (*DomainConfigRecord, error) {
			return a.GetConfigRecord(ctx, domainID)
		},
		// The configuration body carries no domain ID. ID is used only for
		// Name references, which this collection does not support.
		ID: func(*DomainConfigRecord) string { return "" },
	})
}
