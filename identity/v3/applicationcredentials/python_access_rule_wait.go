package applicationcredentials

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// AccessRuleResources returns a Collection over the access rules of one fixed
// user, bound to direct GET users/{user_id}/access_rules/{id} for the shared
// wait engine, such as identity.WaitForDelete. Only resource.ID references are
// supported; status waits require resource.WithStatusAttribute.
func (a *API) AccessRuleResources(userID string) *resource.Collection[AccessRule] {
	return resource.NewCollection(resource.Adapter[AccessRule]{
		Kind: "accessrules",
		Get: func(ctx context.Context, id string) (*AccessRule, error) {
			return a.GetAccessRule(ctx, userID, id)
		},
		ID: func(v *AccessRule) string { return v.ID },
	})
}
