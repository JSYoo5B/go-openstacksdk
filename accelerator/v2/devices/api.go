// Package devices manages Cyborg physical accelerator devices.
package devices

import (
	"encoding/json"
	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/accelerator/v2/common"
	"gophercloudsdk/internal/cyborg"
	"gophercloudsdk/resource"
)

type Device struct {
	common.Metadata
	UUID              string      `json:"uuid"`
	ID                json.Number `json:"id"`
	Hostname          string      `json:"hostname"`
	Type              string      `json:"type"`
	Vendor            string      `json:"vendor"`
	Model             string      `json:"model"`
	Status            string      `json:"status"`
	StandardBoardInfo string      `json:"std_board_info"`
	VendorBoardInfo   string      `json:"vendor_board_info"`
}

func (v *Device) UnmarshalJSON(data []byte) error {
	type plain Device
	var decoded plain
	if err := common.Decode(data, &decoded, &decoded.Metadata); err != nil {
		return err
	}
	*v = Device(decoded)
	return nil
}

type API struct {
	*resource.Collection[Device]
	Resources *resource.Collection[Device]
	client    *gophercloud.ServiceClient
}

func New(client *gophercloud.ServiceClient) *API {
	var status func(*Device) string
	if cyborg.RequireMicroversion(client, 3) == nil {
		status = func(v *Device) string { return v.Status }
	}
	collection := cyborg.Collection(client, "devices", "device", "devices", func(v *Device) string { return v.UUID }, nil, status, func(v *Device) *common.Metadata { return &v.Metadata }, nil)
	return &API{Collection: collection, Resources: collection, client: client}
}

func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }
