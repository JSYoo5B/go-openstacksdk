package zones

import (
	"context"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/dns/v2/zones"
)

// DeleteWithShares issues DELETE /zones/{id} like Delete and, when
// deleteShares is true, adds X-Designate-Delete-Shares: true so Designate also
// removes the zone's shares. When deleteShares is false the header is omitted,
// so the request is identical to Delete. Only 202 is accepted and the
// acknowledged zone is returned; the deletion completes asynchronously. The
// uncertain DELETE is never replayed by the SDK.
func (a *API) DeleteWithShares(ctx context.Context, zoneID string, deleteShares bool) (*Zone, error) {
	var r upstream.DeleteResult
	opts := &gophercloud.RequestOpts{OkCodes: []int{http.StatusAccepted}, JSONResponse: &r.Body}
	if deleteShares {
		opts.MoreHeaders = map[string]string{"X-Designate-Delete-Shares": "true"}
	}
	resp, err := a.client.Delete(ctx, a.client.ServiceURL("zones", zoneID), opts)
	_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
	value, err := r.Extract()
	return value, request.Wrap("DeleteWithShares", "zones", err)
}
