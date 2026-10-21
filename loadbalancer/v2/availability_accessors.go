package v2

import (
	"github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/availabilityzoneprofiles"
	"github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/availabilityzones"
)

// AvailabilityZones returns the SDK-owned Octavia availability zone API bound
// to this service's client. Gophercloud v2.15.0 has no binding for these routes.
func (s *Service) AvailabilityZones() *availabilityzones.API {
	return availabilityzones.New(s.client)
}

// AvailabilityZoneProfiles returns the SDK-owned Octavia availability zone
// profile API bound to this service's client.
func (s *Service) AvailabilityZoneProfiles() *availabilityzoneprofiles.API {
	return availabilityzoneprofiles.New(s.client)
}
