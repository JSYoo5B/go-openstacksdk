package blockstorage

import (
	"context"
	"github.com/JSYoo5B/gophercloudsdk/internal/cloudlimits"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// GetVolumeLimitsRequest optionally selects another project by exact name/ID.
// Empty NameOrID reads the authenticated project's unfiltered limits.
type GetVolumeLimitsRequest = cloudlimits.Input
type GetVolumeLimitsOpts = cloudlimits.Options
type GetVolumeLimitsOption = cloudlimits.Option

// GetVolumeLimitsPage independently owns an admitted physical response.
type GetVolumeLimitsPage = cloudlimits.Page

// GetVolumeLimitsProjectResult keeps project resolution and its original proof.
// SeededID identifies source-compatible member lookup ID fallback, not a field
// that the server supplied. Project always retains the actual raw row fields.
type GetVolumeLimitsProjectResult = cloudlimits.ProjectResult

// GetVolumeLimitsResult keeps project resolution, limits observation and the
// complete nullable logical view separately. RequestedProjectID is a query
// filter, not confirmed ownership of the limits returned by Cinder.
type GetVolumeLimitsResult = cloudlimits.Result

func WithGetVolumeLimitsOptions(value GetVolumeLimitsOpts) GetVolumeLimitsOption {
	return cloudlimits.WithOptions(value)
}

func WithGetVolumeLimitsLocation(value resource.CloudLocation) GetVolumeLimitsOption {
	return cloudlimits.WithLocation(value)
}

func PrepareGetVolumeLimitsOptions(ctx context.Context, options ...GetVolumeLimitsOption) (GetVolumeLimitsOpts, error) {
	return cloudlimits.Prepare(ctx, options, nil)
}
