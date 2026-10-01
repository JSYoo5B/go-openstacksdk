// Package deployables manages Cyborg logical acceleration units.
package deployables

import (
	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/accelerator/v2/common"
	"gophercloudsdk/internal/cyborg"
	"gophercloudsdk/resource"
)

type Deployable struct {
	common.Metadata
	UUID                 string  `json:"uuid"`
	Name                 string  `json:"name"`
	DeviceID             int64   `json:"device_id"`
	ParentID             *int64  `json:"parent_id"`
	RootID               *int64  `json:"root_id"`
	NumAccelerators      int     `json:"num_accelerators"`
	AttributesList       string  `json:"attributes_list"`
	ResourceProviderUUID string  `json:"rp_uuid"`
	DriverName           string  `json:"driver_name"`
	BitstreamID          *string `json:"bitstream_id"`
}

func (v *Deployable) UnmarshalJSON(data []byte) error {
	type plain Deployable
	var decoded plain
	if err := common.Decode(data, &decoded, &decoded.Metadata); err != nil {
		return err
	}
	*v = Deployable(decoded)
	return nil
}

type API struct {
	*resource.Collection[Deployable]
	Resources *resource.Collection[Deployable]
	client    *gophercloud.ServiceClient
}

func New(client *gophercloud.ServiceClient) *API {
	collection := cyborg.Collection(client, "deployables", "deployable", "deployables", func(v *Deployable) string { return v.UUID }, func(v *Deployable) string { return v.Name }, nil, func(v *Deployable) *common.Metadata { return &v.Metadata }, nil)
	return &API{Collection: collection, Resources: collection, client: client}
}

func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }
