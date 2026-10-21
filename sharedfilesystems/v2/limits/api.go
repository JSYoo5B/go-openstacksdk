// Package limits owns Manila's GET /limits, absent from Gophercloud v2.15.0.
// It returns the caller's absolute project limits and usage plus the rate
// limit list, with the actual response retained.
package limits

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/manilaown"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type Evidence = manilaown.Evidence

// Absolute holds the documented counters. Absent or null members are nil;
// Limits.Evidence.Body keeps any other member the service returns.
type Absolute struct {
	MaxTotalReplicaGigabytes   *int64 `json:"maxTotalReplicaGigabytes"`
	MaxTotalShares             *int64 `json:"maxTotalShares"`
	MaxTotalShareGigabytes     *int64 `json:"maxTotalShareGigabytes"`
	MaxTotalShareNetworks      *int64 `json:"maxTotalShareNetworks"`
	MaxTotalShareSnapshots     *int64 `json:"maxTotalShareSnapshots"`
	MaxTotalShareReplicas      *int64 `json:"maxTotalShareReplicas"`
	MaxTotalSnapshotGigabytes  *int64 `json:"maxTotalSnapshotGigabytes"`
	TotalReplicaGigabytesUsed  *int64 `json:"totalReplicaGigabytesUsed"`
	TotalShareGigabytesUsed    *int64 `json:"totalShareGigabytesUsed"`
	TotalSharesUsed            *int64 `json:"totalSharesUsed"`
	TotalShareNetworksUsed     *int64 `json:"totalShareNetworksUsed"`
	TotalShareSnapshotsUsed    *int64 `json:"totalShareSnapshotsUsed"`
	TotalSnapshotGigabytesUsed *int64 `json:"totalSnapshotGigabytesUsed"`
	TotalShareReplicasUsed     *int64 `json:"totalShareReplicasUsed"`
}

// Limits is the typed view of the "limits" object. Rate rows stay raw JSON.
type Limits struct {
	Absolute Absolute          `json:"absolute"`
	Rate     []json.RawMessage `json:"rate"`
	Evidence Evidence          `json:"-"`
}

type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }

func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }

// GetOpts carries query extensions only; Manila's limits route has no filter.
type GetOpts struct{}

type GetOption = request.Option[GetOpts]

func WithQuery(key, value string) GetOption { return request.WithQuery[GetOpts](key, value) }

// Get reads GET /limits once; only HTTP 200 succeeds.
func (a *API) Get(ctx context.Context, options ...GetOption) (*Limits, error) {
	wrap := func(err error) error { return request.Wrap("Get", "limits", err) }
	if a == nil || a.client == nil {
		return nil, wrap(fmt.Errorf("%w: limits API is required", resource.ErrInvalidOption))
	}
	if err := ctx.Err(); err != nil {
		return nil, wrap(err)
	}
	cfg, err := request.Apply(GetOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(cfg, false, true, false)
	}
	if err != nil {
		return nil, wrap(err)
	}
	resp, err := manilaown.Call{Client: a.client, Method: http.MethodGet, URL: manilaown.WithQuery(a.client.ServiceURL("limits"), cfg.Query), OK: []int{http.StatusOK}}.Do(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	value, err := manilaown.One[Limits](resp, "limits")
	return value, wrap(err)
}
