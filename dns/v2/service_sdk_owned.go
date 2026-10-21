package v2

import (
	"github.com/JSYoo5B/go-openstacksdk/dns/v2/blacklists"
	"github.com/JSYoo5B/go-openstacksdk/dns/v2/limits"
	"github.com/JSYoo5B/go-openstacksdk/dns/v2/ptr/floatingips"
	"github.com/JSYoo5B/go-openstacksdk/dns/v2/servicestatuses"
	"github.com/JSYoo5B/go-openstacksdk/dns/v2/tlds"
	"github.com/JSYoo5B/go-openstacksdk/dns/v2/zoneexports"
	"github.com/JSYoo5B/go-openstacksdk/dns/v2/zoneimports"
)

// Gophercloud v2.15.0 has no binding for these Designate resources, so the SDK
// owns them. Each accessor is cheap: it binds the shared service client.

// Blacklists returns the /blacklists API.
func (s *Service) Blacklists() *blacklists.API { return blacklists.New(s.client) }

// TLDs returns the /tlds API.
func (s *Service) TLDs() *tlds.API { return tlds.New(s.client) }

// ZoneExports returns the zone export task API.
func (s *Service) ZoneExports() *zoneexports.API { return zoneexports.New(s.client) }

// ZoneImports returns the zone import task API.
func (s *Service) ZoneImports() *zoneimports.API { return zoneimports.New(s.client) }

// FloatingIPPTR returns the /reverse/floatingips PTR record API.
func (s *Service) FloatingIPPTR() *floatingips.API { return floatingips.New(s.client) }

// Limits returns the /limits API.
func (s *Service) Limits() *limits.API { return limits.New(s.client) }

// ServiceStatuses returns the /service_statuses API.
func (s *Service) ServiceStatuses() *servicestatuses.API { return servicestatuses.New(s.client) }
