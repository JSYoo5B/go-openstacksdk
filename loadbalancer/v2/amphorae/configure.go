package amphorae

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
)

// Configure asks Octavia to push the current configuration to one amphora
// agent (PUT /octavia/amphorae/{id}/config). The request has no body and only
// 202 is accepted. The pinned Gophercloud release has no native binding for
// this route, so the call goes through the service client directly.
//
// Acknowledgement is not completion: the amphora is reconfigured
// asynchronously and callers must observe its status explicitly. The call is
// never replayed automatically and cancelling the context does not roll back
// remote work.
func (a *API) Configure(ctx context.Context, id string) error {
	resp, err := a.client.Put(ctx, a.client.ServiceURL("octavia", "amphorae", id, "config"), nil, nil, &gophercloud.RequestOpts{
		OkCodes: []int{202},
	})
	_, _, err = gophercloud.ParseResponse(resp, err)
	return request.Wrap("Configure", "amphorae", err)
}
