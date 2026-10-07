package blockstorage

import (
	"context"
	"github.com/JSYoo5B/gophercloudsdk/internal/cloudlimits"
	"github.com/gophercloud/gophercloud/v2"
)

// GetVolumeLimits reads selected Cinder v3 limits. A nonempty project identity
// is resolved first through Identity v3 even if it resembles an ID. Empty input
// skips Identity and uses Cinder's unfiltered authenticated-project behavior.
// Selected microversions are preserved without a project-filter preflight.
func GetVolumeLimits(ctx context.Context, cinder, identity *gophercloud.ServiceClient, input GetVolumeLimitsRequest, options ...GetVolumeLimitsOption) (*GetVolumeLimitsResult, error) {
	return cloudlimits.ReadSelected(ctx, cinder, identity, input, options)
}
