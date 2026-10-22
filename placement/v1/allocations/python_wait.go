package allocations

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Allocation is one consumer's allocation set. ConsumerUUID is the polled
// reference, because the response body does not carry it.
type Allocation struct {
	ConsumerUUID string
	*Allocations
}

// Resources returns a Collection bound to direct GET allocations/{consumer}
// for the shared wait engine, such as placement.WaitForStatus with
// resource.WithStatusAttribute and placement.WaitForDelete. Only resource.ID
// references are supported; listing, name lookup and deletion stay on API.
// Placement answers GET for a consumer without allocations with an empty
// allocation set rather than 404, so a deletion wait observes only what the
// service actually returns and normally reaches its deadline, as Python
// wait_for_delete does for Allocation.
func (a *API) Resources() *resource.Collection[Allocation] {
	return resource.NewCollection(resource.Adapter[Allocation]{
		Kind: "allocations",
		Get: func(ctx context.Context, consumer string) (*Allocation, error) {
			value, err := a.Get(ctx, consumer)
			if err != nil || value == nil {
				return nil, err
			}
			return &Allocation{ConsumerUUID: consumer, Allocations: value}, nil
		},
		ID: func(v *Allocation) string { return v.ConsumerUUID },
	})
}
