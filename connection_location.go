package openstack

import (
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudlocation"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// WithCloudLocation supplies owned configured facts when an adopted provider
// cannot recover cloud/authentication configuration. It replaces the complete
// location, including the project scope ID, and is accepted by FromProvider.
func WithCloudLocation(value resource.CloudLocation) ConnectionOption {
	owned := value.Clone()
	return func(options *connectionOptions) error {
		copy := owned.Clone()
		options.cloudLocation = &copy
		return nil
	}
}

// CurrentLocation snapshots cloud, region and recorded token scope without
// making an API or authentication request. Unknown facts remain null.
func (c *Connection) CurrentLocation() (resource.CloudLocation, error) {
	if c == nil || c.provider == nil {
		return resource.CloudLocation{}, invalid("Connection is required")
	}
	if c.options.cloudLocation != nil {
		return c.options.cloudLocation.Clone(), nil
	}
	value := c.options.locationFacts.Clone()
	id, err := cloudlocation.ProjectID(c.provider)
	if err != nil {
		return resource.CloudLocation{}, err
	}
	value.Project.ID = id
	return value, nil
}

func configuredLocation(cloudName string, auth gophercloud.AuthOptions) resource.CloudLocation {
	var value resource.CloudLocation
	if cloudName != "" {
		value.Cloud = locationString(cloudName)
	} else if parsed, err := url.Parse(auth.IdentityEndpoint); err == nil && parsed.Hostname() != "" {
		value.Cloud = locationString(parsed.Hostname())
	}
	value.Project.Name = locationNonempty(auth.TenantName)
	if scope := auth.Scope; scope != nil && (scope.ProjectName != "" || scope.ProjectID != "") {
		if scope.ProjectName != "" {
			value.Project.Name = locationString(scope.ProjectName)
		}
		value.Project.DomainID = locationNonempty(scope.DomainID)
		value.Project.DomainName = locationNonempty(scope.DomainName)
	} else if auth.TenantName != "" {
		value.Project.DomainID = locationNonempty(auth.DomainID)
		value.Project.DomainName = locationNonempty(auth.DomainName)
	}
	return value
}

func locationString(value string) *string { copy := value; return &copy }
func locationNonempty(value string) *string {
	if value == "" {
		return nil
	}
	return locationString(value)
}
