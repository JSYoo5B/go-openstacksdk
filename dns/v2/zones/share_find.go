package zones

import (
	"context"
	"iter"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/dns/v2/internal/designate"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/dns/v2/zones"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

// SharesQuery filters a zone's share list. TargetProjectID is Python's
// target_project_id query; AllProjects and SudoProject are the documented
// X-Auth-All-Projects / X-Auth-Sudo-Project-ID headers. Unset fields send
// nothing; an explicit false AllProjects is sent as "false".
type SharesQuery struct {
	TargetProjectID *string
	AllProjects     *bool
	SudoProject     string
}

// ListSharesFiltered lists shares of one zone with the target_project_id filter
// that the native ListShares cannot express. Paging follows the native
// links.next pager.
func (a *API) ListSharesFiltered(ctx context.Context, zoneID string, filter SharesQuery) iter.Seq2[*ZoneShare, error] {
	fail := func(err error) iter.Seq2[*ZoneShare, error] {
		err = request.Wrap("ListSharesFiltered", "zones", err)
		return func(yield func(*ZoneShare, error) bool) { yield(nil, err) }
	}
	if err := designate.ValidateID(zoneID); err != nil {
		return fail(err)
	}
	endpoint := a.client.ServiceURL("zones", url.PathEscape(zoneID), "shares")
	if filter.TargetProjectID != nil {
		endpoint += "?" + url.Values{"target_project_id": {*filter.TargetProjectID}}.Encode()
	}
	pager := pagination.NewPager(a.client, endpoint, func(r pagination.PageResult) pagination.Page {
		return upstream.ZoneSharePage{LinkedPageBase: pagination.LinkedPageBase{PageResult: r}}
	})
	pager.Headers = designate.Headers(filter.AllProjects, filter.SudoProject)
	return resource.Stream(ctx, pager, func(page pagination.Page) ([]ZoneShare, error) {
		values, err := upstream.ExtractZoneShares(page)
		return []ZoneShare(values), err
	})
}

// FindShare is Python find_zone_share: GET /zones/{zone}/shares/{identity}
// first, then a list of the zone's shares matching the identifier against the
// share ID (shares have no name). Absence defaults to nil, nil; use
// resource.WithIdentityFindIgnoreMissing(false) for an error. Duplicate IDs
// fail, and the fallback follows resource.FindIdentity's policy (GET 400/403/404).
func (a *API) FindShare(ctx context.Context, zoneID, identity string, options ...resource.IdentityFindOption) (*ZoneShare, error) {
	if err := designate.ValidateID(zoneID); err != nil {
		return nil, request.Wrap("FindShare", "zones", err)
	}
	shares := resource.NewCollection(resource.Adapter[ZoneShare]{
		Kind:         "zone shares",
		IdentityFind: true,
		Get:          func(ctx context.Context, id string) (*ZoneShare, error) { return a.GetShare(ctx, zoneID, id) },
		ID:           func(v *ZoneShare) string { return v.ID },
		Name:         func(*ZoneShare) string { return "" },
		IdentityName: func(*ZoneShare) string { return "" },
		Iterate: func(ctx context.Context, _ url.Values) iter.Seq2[*ZoneShare, error] {
			return a.ListShares(ctx, zoneID)
		},
	})
	value, err := shares.FindIdentity(ctx, identity, options...)
	return value, request.Wrap("FindShare", "zones", err)
}
