package rules

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Resources returns a Collection bound to direct GET rules/{uuid} for the
// shared wait engine, such as baremetalintrospection/v1.WaitForDelete. Only
// resource.ID references are supported; the inspector has no rule names.
// Rule has no status attribute; status waits require
// resource.WithStatusAttribute with an exported string field.
func (a *API) Resources() *resource.Collection[Rule] {
	return resource.NewCollection(resource.Adapter[Rule]{
		Kind: kind,
		Get:  func(ctx context.Context, id string) (*Rule, error) { return a.Get(ctx, id) },
		ID:   func(v *Rule) string { return v.UUID },
	})
}
