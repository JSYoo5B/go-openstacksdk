package zones

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/dns/v2/zones"
)

// shareZone serializes the builder through ToShareMap. The pinned native Share
// reflects over the builder value itself, which loses the SDK wrapper's typed
// and extension fields. sdkgen audits that native declaration before emitting
// this call; the URL, 201 success code, and result type stay native.
func shareZone(ctx context.Context, client *gophercloud.ServiceClient, zoneID string, opts upstream.ShareOptsBuilder) (r upstream.ZoneShareResult) {
	body, err := opts.ToShareMap()
	if err != nil {
		r.Err = err
		return
	}
	resp, err := client.Post(ctx, client.ServiceURL("zones", zoneID, "shares"), body, &r.Body, &gophercloud.RequestOpts{
		OkCodes: []int{201},
	})
	_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
	return
}
