// Package acceleratorrequests manages Cyborg requests and their binding state.
package acceleratorrequests

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/common"
	"github.com/JSYoo5B/go-openstacksdk/internal/cyborg"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type AcceleratorRequest struct {
	common.Metadata
	UUID                 string            `json:"uuid"`
	State                string            `json:"state"`
	DeviceProfileName    string            `json:"device_profile_name"`
	DeviceProfileGroupID int64             `json:"device_profile_group_id"`
	Hostname             *string           `json:"hostname"`
	DeviceRPUUID         *string           `json:"device_rp_uuid"`
	InstanceUUID         *string           `json:"instance_uuid"`
	ProjectID            *string           `json:"project_id"`
	AttachHandleType     *string           `json:"attach_handle_type"`
	AttachHandleUUID     *string           `json:"attach_handle_uuid"`
	AttachHandleInfo     map[string]string `json:"attach_handle_info"`
}

func (v *AcceleratorRequest) UnmarshalJSON(data []byte) error {
	type plain AcceleratorRequest
	var decoded plain
	if err := common.Decode(data, &decoded, &decoded.Metadata); err != nil {
		return err
	}
	if err := validateID(decoded.UUID); err != nil {
		return fmt.Errorf("Cyborg ARQ response: %w", err)
	}
	*v = AcceleratorRequest(decoded)
	return nil
}

type API struct {
	*resource.Collection[AcceleratorRequest]
	Resources *resource.Collection[AcceleratorRequest]
	client    *gophercloud.ServiceClient
}

func New(client *gophercloud.ServiceClient) *API {
	collection := cyborg.Collection(client, "accelerator_requests", "arq", "arqs",
		func(v *AcceleratorRequest) string { return v.UUID }, nil,
		func(v *AcceleratorRequest) string { return v.State },
		func(v *AcceleratorRequest) *common.Metadata { return &v.Metadata },
		func(id string) string {
			return client.ServiceURL("accelerator_requests") + "?" + url.Values{"arqs": {id}}.Encode()
		}, validateID)
	return &API{Collection: collection, Resources: collection, client: client}
}

func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }

func validateID(id string) error {
	if err := resource.ID(id).Validate(); err != nil {
		return err
	}
	if strings.ContainsAny(id, ",\x00") {
		return fmt.Errorf("%w: accelerator request ID must identify one request", resource.ErrInvalidOption)
	}
	return nil
}
