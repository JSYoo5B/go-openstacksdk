// Package limits binds Designate's API limits (GET /v2/limits).
package limits

import (
	"context"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/dns/v2/internal/designate"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
)

// Limits is the decoded flat limits object. Body retains omitted and null
// values and any additional server fields.
type Limits struct {
	designate.Evidence
	MaxPageLimit           int `json:"max_page_limit"`
	MaxRecordsetNameLength int `json:"max_recordset_name_length"`
	MaxRecordsetRecords    int `json:"max_recordset_records"`
	MaxZoneNameLength      int `json:"max_zone_name_length"`
	MaxZoneRecords         int `json:"max_zone_records"`
	MaxZoneRecordsets      int `json:"max_zone_recordsets"`
	MaxZones               int `json:"max_zones"`
	MinTTL                 int `json:"min_ttl"`
}

// API owns one authenticated Designate client.
type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API     { return &API{client: client} }
func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }

// Get sends GET /limits and accepts only 200. Python limits() yields exactly
// this one object; there is no collection or paging.
func (a *API) Get(ctx context.Context) (*Limits, error) {
	if err := designate.Client(ctx, a.client); err != nil {
		return nil, request.Wrap("Get", "limits", err)
	}
	response, err := designate.Do(ctx, a.client, http.MethodGet, a.client.ServiceURL("limits"), nil, nil, http.StatusOK)
	if err != nil {
		return nil, request.Wrap("Get", "limits", err)
	}
	value, err := designate.Decode[Limits](response)
	return value, request.Wrap("Get", "limits", err)
}
