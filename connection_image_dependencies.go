package openstack

import (
	imageapi "github.com/JSYoo5B/go-openstacksdk/image/v2"
	"github.com/gophercloud/gophercloud/v2"
)

// The versioned and high-level image entry points share Connection facts as
// well as their authenticated service client.
func (c *Connection) newImageV2Service(client *gophercloud.ServiceClient) *imageapi.Service {
	return imageapi.NewWithDependencies(client, imageapi.Dependencies{CloudLocation: c.CurrentLocation})
}
