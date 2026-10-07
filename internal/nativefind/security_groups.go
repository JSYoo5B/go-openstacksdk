package nativefind

import (
	"context"
	"iter"
	"net/url"

	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

// IterateSecurityGroups owns the raw query for the audited Neutron collection.
// The native List accepts a concrete struct, so this pager retains repeated and
// explicitly empty wire values without manufacturing a caller builder. Native
// SecGroupPage extraction, status codes, continuation and whole-page errors are
// unchanged, as are the original service/provider clients. Row/page controls
// are local and do not add a wire limit.
func IterateSecurityGroups(ctx context.Context, client *gophercloud.ServiceClient, values url.Values, control resource.ListControl) iter.Seq2[*groups.SecGroup, error] {
	frozen := cloneListQuery(values)
	return func(yield func(*groups.SecGroup, error) bool) {
		target, err := nativeListURL(ctx, client, []string{"security-groups"})
		if err != nil {
			yield(nil, err)
			return
		}
		if query := frozen.Encode(); query != "" {
			target += "?" + query
		}
		pager := pagination.NewPager(client, target, func(result pagination.PageResult) pagination.Page {
			return groups.SecGroupPage{LinkedPageBase: pagination.LinkedPageBase{PageResult: result}}
		})
		resource.StreamWithControl(ctx, pager, groups.ExtractGroups, control)(yield)
	}
}

// IterateSecurityGroupBodies pairs native SecurityGroup models with owned
// original response fields. It retains the existing raw-query pager's source,
// status, continuation and whole-page decoding policy. Controls count native
// rows before the collection evaluates local Body conditions.
func IterateSecurityGroupBodies(ctx context.Context, client *gophercloud.ServiceClient, values url.Values, control resource.ListControl) iter.Seq2[*resource.BodyRecord[groups.SecGroup], error] {
	frozen := cloneListQuery(values)
	return func(yield func(*resource.BodyRecord[groups.SecGroup], error) bool) {
		target, err := nativeListURL(ctx, client, []string{"security-groups"})
		if err != nil {
			yield(nil, err)
			return
		}
		if query := frozen.Encode(); query != "" {
			target += "?" + query
		}
		pager := pagination.NewPager(client, target, func(result pagination.PageResult) pagination.Page {
			return groups.SecGroupPage{LinkedPageBase: pagination.LinkedPageBase{PageResult: result}}
		})
		resource.BodyStreamWithControl(ctx, pager, groups.ExtractGroups, "security_groups", control)(yield)
	}
}
