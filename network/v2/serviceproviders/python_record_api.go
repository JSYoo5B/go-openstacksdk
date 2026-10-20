// Package serviceproviders implements the owned Neutron service provider list.
package serviceproviders

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// API shares a selected Network service client and owns the Source contracts.
type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }
func (a *API) RawClient() *gophercloud.ServiceClient {
	if a == nil {
		return nil
	}
	return a.client
}

// ServiceProviderRecord separates the normalized view from actual evidence.
type ServiceProviderRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	row            json.RawMessage
	fromList       bool
}

func findInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

var serviceProviderProfile = neutronfind.Profile{
	Kind: "service_providers", Path: "service-providers", Plural: "service_providers", IDKey: "id", NameKey: "name", VersionedPath: "/v2.0/service-providers", PreserveUnknownBody: true,
	Body:  []neutronfind.Descriptor{{Name: "id", Wire: "id", Missing: "null"}, {Name: "name", Wire: "name", Missing: "null"}, {Name: "service_type", Wire: "service_type", Missing: "null"}, {Name: "is_default", Wire: "default", Kind: neutronfind.Boolean, Missing: "null"}},
	Query: []neutronfind.QueryPair{{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}, {Name: "fields", Wire: "fields"}, {Name: "service_type", Wire: "service_type"}, {Name: "name", Wire: "name"}, {Name: "is_default", Wire: "default"}},
}
