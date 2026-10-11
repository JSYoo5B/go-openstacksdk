package volumes

import (
	"context"
	"fmt"
	"iter"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

// ListWithDetails selects Python volumes(details=...). List keeps its existing
// detail default; false uses the summary /volumes route and the same options.
func (a *API) ListWithDetails(ctx context.Context, details bool, options ...ListOption) iter.Seq2[*Volume, error] {
	if details {
		return a.List(ctx, options...)
	}
	return a.ListSummary(ctx, options...)
}

// ListSummary reads /volumes rather than /volumes/detail. It retains the
// existing concrete query options, native Volume decoder and linked pager.
func (a *API) ListSummary(ctx context.Context, options ...ListOption) iter.Seq2[*Volume, error] {
	return a.ListSummaryWithControl(ctx, resource.ListControl{}, options...)
}

// ListSummaryWithControl additionally applies the shared local raw-row cap and
// first-page policy. The control never overwrites the caller's wire limit.
func (a *API) ListSummaryWithControl(ctx context.Context, control resource.ListControl, options ...ListOption) iter.Seq2[*Volume, error] {
	failure := func(err error) iter.Seq2[*Volume, error] {
		return func(yield func(*Volume, error) bool) {
			yield(nil, request.Wrap("ListSummary", "volumes", cloudread.ContextError(ctx, err)))
		}
	}
	if ctx == nil || a == nil || a.client == nil || a.client.ProviderClient == nil {
		return failure(fmt.Errorf("%w: summary listing requires context and configured volume API", resource.ErrInvalidOption))
	}
	if err := cloudread.Context(ctx); err != nil {
		return failure(err)
	}
	cfg, err := request.Apply(ListOpts{}, options...)
	if err != nil {
		return failure(err)
	}
	if err := cloudread.Context(ctx); err != nil {
		return failure(err)
	}
	if err := request.ValidateCapabilities(cfg, false, true, false); err != nil {
		return failure(err)
	}
	builder := listOptsBuilder{base: cfg.Options, config: cfg}
	query, err := builder.ToVolumeListQuery()
	if err != nil {
		return failure(err)
	}
	pager := pagination.NewPager(a.client, a.client.ServiceURL("volumes")+query, func(result pagination.PageResult) pagination.Page {
		return upstream.VolumePage{LinkedPageBase: pagination.LinkedPageBase{PageResult: result}}
	})
	sequence := resource.StreamWithControl(ctx, pager, upstream.ExtractVolumes, control)
	return func(yield func(*Volume, error) bool) {
		for value, err := range sequence {
			if !yield(value, request.Wrap("ListSummary", "volumes", cloudread.ContextError(ctx, err))) {
				return
			}
		}
	}
}
