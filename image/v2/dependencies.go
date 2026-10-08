package v2

import (
	"github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties"
	"github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Dependencies provides Connection facts for owned resource records.
type Dependencies struct {
	CloudLocation func() (resource.CloudLocation, error)
}

func NewWithDependencies(client *gophercloud.ServiceClient, dependencies Dependencies) *Service {
	service := New(client)
	service.MetadefResourceTypes = metadefresourcetypes.NewWithDependencies(client, metadefresourcetypes.Dependencies{CloudLocation: dependencies.CloudLocation})
	service.MetadefProperties = metadefproperties.NewWithDependencies(client, metadefproperties.Dependencies{CloudLocation: dependencies.CloudLocation})
	return service
}
