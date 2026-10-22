package ruletypes

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Resources returns a Collection bound to direct GET qos/rule-types/{type}
// for the shared wait engine, such as network.WaitForDelete. The reference is
// the rule type name; only resource.ID references are supported. RuleType has
// no status attribute; status waits require resource.WithStatusAttribute.
func (a *API) Resources() *resource.Collection[RuleType] {
	return resource.NewCollection(resource.Adapter[RuleType]{
		Kind: "ruletypes",
		Get:  func(ctx context.Context, name string) (*RuleType, error) { return a.GetRuleType(ctx, name) },
		ID:   func(v *RuleType) string { return v.Type },
	})
}
