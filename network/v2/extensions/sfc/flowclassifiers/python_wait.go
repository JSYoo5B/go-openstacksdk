package flowclassifiers

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Resources returns a Collection bound to direct GET sfc/flow_classifiers/{id} for the shared
// wait engine, such as network.WaitForStatus and network.WaitForDelete. Only
// resource.ID references are supported; listing, name lookup and deletion stay
// on API. FlowClassifier has no status attribute; status waits require resource.WithStatusAttribute with an exported string field.
func (a *API) Resources() *resource.Collection[FlowClassifier] {
	return resource.NewCollection(resource.Adapter[FlowClassifier]{
		Kind: "flowclassifiers",
		Get:  func(ctx context.Context, id string) (*FlowClassifier, error) { return a.Get(ctx, id) },
		ID:   func(v *FlowClassifier) string { return v.ID },
	})
}
