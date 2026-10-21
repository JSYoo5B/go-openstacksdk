package sharenetworks

import (
	"context"
	"iter"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/sharenetworks"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

// ListOption configures List.
type ListOption = request.Option[ListOpts]

// WithListOptions replaces the typed List filters.
func WithListOptions(value ListOpts) ListOption { return request.WithOptions(value) }

// WithListQuery adds one raw query parameter, replacing a typed filter with
// the same name.
func WithListQuery(key, value string) ListOption { return request.WithQuery[ListOpts](key, value) }

type listOptsBuilder struct {
	base   ListOpts
	config request.Config[ListOpts]
}

func (b listOptsBuilder) ToShareNetworkListQuery() (string, error) {
	query, err := b.base.ToShareNetworkListQuery()
	if err != nil {
		return "", err
	}
	return request.ExtendQuery(query, b.config.Query)
}

// List streams the summary view of share networks from GET /share-networks
// (id, name and links only on current Manila releases) with the same filters
// and offset paging as ListDetail. It never requests /share-networks/detail.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*ShareNetwork, error] {
	fail := func(err error) iter.Seq2[*ShareNetwork, error] {
		err = request.Wrap("List", "sharenetworks", err)
		return func(yield func(*ShareNetwork, error) bool) { yield(nil, err) }
	}
	cfg, err := request.Apply(ListOpts{}, options...)
	if err != nil {
		return fail(err)
	}
	if err = request.ValidateCapabilities(cfg, false, true, false); err != nil {
		return fail(err)
	}
	query, err := listOptsBuilder{base: cfg.Options, config: cfg}.ToShareNetworkListQuery()
	if err != nil {
		return fail(err)
	}
	pager := pagination.NewPager(a.client, a.client.ServiceURL("share-networks")+query, func(r pagination.PageResult) pagination.Page {
		p := upstream.ShareNetworkPage{MarkerPageBase: pagination.MarkerPageBase{PageResult: r}}
		p.Owner = p
		return p
	})
	return resource.StreamWithControl(ctx, pager, func(page pagination.Page) ([]ShareNetwork, error) {
		values, err := upstream.ExtractShareNetworks(page)
		return []ShareNetwork(values), err
	}, resource.ListControl{})
}
