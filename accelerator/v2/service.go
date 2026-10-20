// Package accelerator provides the SDK-owned Cyborg v2 service.
package accelerator

import (
	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/acceleratorrequests"
	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/attributes"
	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/deployables"
	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/deviceprofiles"
	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/devices"
	"github.com/gophercloud/gophercloud/v2"
)

type Service struct {
	client              *gophercloud.ServiceClient
	Deployables         *deployables.API
	Devices             *devices.API
	AcceleratorRequests *acceleratorrequests.API
	DeviceProfiles      *deviceprofiles.API
	Attributes          *attributes.API
}

func New(client *gophercloud.ServiceClient) *Service {
	return &Service{client: client, Deployables: deployables.New(client), Devices: devices.New(client), AcceleratorRequests: acceleratorrequests.New(client), DeviceProfiles: deviceprofiles.New(client), Attributes: attributes.New(client)}
}

func (s *Service) RawClient() *gophercloud.ServiceClient { return s.client }
