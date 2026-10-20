package blockstorage

import (
	"context"
	"net/url"

	"github.com/gophercloud/gophercloud/v2"
)

// captureCinderReader owns the selected source before any caller option runs.
// Resource bindings provide their fixed collection path and raw decoding policy.
func captureCinderReader(ctx context.Context, cinder *gophercloud.ServiceClient, path ...string) (*preparedGetVolumes, error) {
	if err := attachContext(ctx); err != nil {
		return nil, err
	}
	source, err := captureAttachSource(ctx, cinder, "volume")
	if err != nil {
		return nil, err
	}
	if source.client.Type == "" {
		source.client.Type = "volumev3"
	}
	target := source.client.ServiceURL(path...)
	if err := validateAttachTarget(&source.client, target); err != nil {
		return nil, err
	}
	initial, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	p := &preparedGetVolumes{cinder: source, initialURL: initial}
	return p, p.guard(ctx)
}
