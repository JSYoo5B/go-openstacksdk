package zones

import (
	"context"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
)

// Nameserver is one authoritative nameserver of a zone's pool.
type Nameserver struct {
	Hostname string `json:"hostname"`
	Priority int    `json:"priority"`
}

// Abandon issues POST /zones/{id}/tasks/abandon without a request body. The
// zone is removed from Designate without touching the backend DNS servers.
// Designate answers 202 or 204; any other status is an error. The uncertain
// POST is never replayed and a cancelled context does not roll back remote work.
func (a *API) Abandon(ctx context.Context, zoneID string) error {
	return a.zoneTask(ctx, "Abandon", zoneID, "abandon")
}

// Xfr issues POST /zones/{id}/tasks/xfr without a request body, asking a
// SECONDARY zone to refresh from its masters. The 202 acknowledgement is not
// completion of the transfer.
func (a *API) Xfr(ctx context.Context, zoneID string) error {
	return a.zoneTask(ctx, "Xfr", zoneID, "xfr")
}

func (a *API) zoneTask(ctx context.Context, op, zoneID, task string) error {
	resp, err := a.client.Post(ctx, a.client.ServiceURL("zones", zoneID, "tasks", task), nil, nil, &gophercloud.RequestOpts{
		OkCodes: []int{http.StatusAccepted, http.StatusNoContent},
	})
	_, _, err = gophercloud.ParseResponse(resp, err)
	return request.Wrap(op, "zones", err)
}

// Nameservers issues GET /zones/{id}/nameservers and returns the nameservers
// in response order. Only 200 is accepted; unknown response fields are not
// retained.
func (a *API) Nameservers(ctx context.Context, zoneID string) ([]Nameserver, error) {
	var body struct {
		Nameservers []Nameserver `json:"nameservers"`
	}
	resp, err := a.client.Get(ctx, a.client.ServiceURL("zones", zoneID, "nameservers"), &body, &gophercloud.RequestOpts{
		OkCodes: []int{http.StatusOK},
	})
	_, _, err = gophercloud.ParseResponse(resp, err)
	if err != nil {
		return nil, request.Wrap("Nameservers", "zones", err)
	}
	if body.Nameservers == nil {
		body.Nameservers = []Nameserver{}
	}
	return body.Nameservers, nil
}
