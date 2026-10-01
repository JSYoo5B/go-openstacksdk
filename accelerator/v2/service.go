// Package accelerator provides the SDK-owned Cyborg v2 service.
package accelerator

import (
	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/accelerator/v2/deployables"
	"gophercloudsdk/accelerator/v2/devices"
)

type Service struct {
	client      *gophercloud.ServiceClient
	Deployables *deployables.API
	Devices     *devices.API
}

func New(client *gophercloud.ServiceClient) *Service {
	return &Service{client: client, Deployables: deployables.New(client), Devices: devices.New(client)}
}

func (s *Service) RawClient() *gophercloud.ServiceClient { return s.client }
