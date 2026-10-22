package localipassociations

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Resources returns a Collection over the port associations of one fixed local
// IP, bound to direct GET local_ips/{local_ip_id}/port_associations/{id} for
// the shared wait engine, such as network.WaitForDelete. Only resource.ID
// references are supported. LocalIPAssociation has no status attribute; status
// waits require resource.WithStatusAttribute with an exported string field.
func (a *API) Resources(localIPID string) *resource.Collection[LocalIPAssociation] {
	return resource.NewCollection(resource.Adapter[LocalIPAssociation]{
		Kind: "localipassociations",
		Get: func(ctx context.Context, id string) (*LocalIPAssociation, error) {
			return a.Get(ctx, localIPID, id)
		},
		ID: func(v *LocalIPAssociation) string { return v.ID },
	})
}
