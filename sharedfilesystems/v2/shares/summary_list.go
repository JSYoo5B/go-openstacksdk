package shares

import (
	"context"
	"iter"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/manilaown"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/shares"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

// ListOption is one option of List: WithListOptions, request.Query (a string
// or []string value; it replaces a typed filter with the same name) or
// request.Custom over *request.Config[ListOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type ListOption = request.ListCall

// WithListOptions replaces the typed List filters.
func WithListOptions(value ListOpts) ListOption {
	return request.Custom(request.WithOptions(value))
}

type listOptsBuilder struct {
	base   ListOpts
	config request.Config[ListOpts]
}

func (b listOptsBuilder) ToShareListQuery() (string, error) {
	query, err := b.base.ToShareListQuery()
	if err != nil {
		return "", err
	}
	return request.ExtendQuery(query, b.config.Query)
}

// List streams the summary view of shares from GET /shares (id, name and
// links only on current Manila releases). It uses the same filters and offset
// paging as ListDetail but never requests /shares/detail. Use ListDetail for
// complete Share models.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*Share, error] {
	fail := func(err error) iter.Seq2[*Share, error] {
		err = request.Wrap("List", "shares", err)
		return func(yield func(*Share, error) bool) { yield(nil, err) }
	}
	cfg, err := calls.ApplyConfig(ListOpts{}, manilaown.QueryCalls[ListOpts](), options...)
	if err != nil {
		return fail(err)
	}
	if err = request.ValidateCapabilities(cfg, false, true, false); err != nil {
		return fail(err)
	}
	query, err := listOptsBuilder{base: cfg.Options, config: cfg}.ToShareListQuery()
	if err != nil {
		return fail(err)
	}
	pager := pagination.NewPager(a.client, a.client.ServiceURL("shares")+query, func(r pagination.PageResult) pagination.Page {
		p := upstream.SharePage{MarkerPageBase: pagination.MarkerPageBase{PageResult: r}}
		p.Owner = p
		return p
	})
	return resource.StreamWithControl(ctx, pager, func(page pagination.Page) ([]Share, error) {
		values, err := upstream.ExtractShares(page)
		return []Share(values), err
	}, resource.ListControl{})
}
