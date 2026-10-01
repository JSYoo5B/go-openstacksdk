// Package accelerator provides the SDK-owned Cyborg v2 service.
package accelerator

import (
	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/accelerator/v2/acceleratorrequests"
	"gophercloudsdk/accelerator/v2/attributes"
	"gophercloudsdk/accelerator/v2/deployables"
	"gophercloudsdk/accelerator/v2/deviceprofiles"
	"gophercloudsdk/accelerator/v2/devices"
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
