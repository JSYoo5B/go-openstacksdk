package v2

import (
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/limits"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/resourcelocks"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/sharegroups"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/sharegroupsnapshots"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shareinstances"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/sharenetworksubnets"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/snapshotinstances"
)

// The SDK-owned packages below cover Manila routes that Gophercloud v2.15.0
// lacks. Each accessor binds the service's authenticated client; the returned
// APIs are stateless and cheap to create.

// Limits returns the GET /limits API.
func (s *Service) Limits() *limits.API { return limits.New(s.client) }

// ResourceLocks returns the resource-lock API (microversion 2.81).
func (s *Service) ResourceLocks() *resourcelocks.API { return resourcelocks.New(s.client) }

// ShareGroups returns the share-group API (microversion 2.55 or later).
func (s *Service) ShareGroups() *sharegroups.API { return sharegroups.New(s.client) }

// ShareGroupSnapshots returns the share-group-snapshot API (2.55 or later).
func (s *Service) ShareGroupSnapshots() *sharegroupsnapshots.API {
	return sharegroupsnapshots.New(s.client)
}

// ShareInstances returns the administrator share-instance API (2.3 or later).
func (s *Service) ShareInstances() *shareinstances.API { return shareinstances.New(s.client) }

// SnapshotInstances returns the snapshot-instance API (2.19 or later).
func (s *Service) SnapshotInstances() *snapshotinstances.API {
	return snapshotinstances.New(s.client)
}

// ShareNetworkSubnets returns the share network subnet API (2.51 or later).
func (s *Service) ShareNetworkSubnets() *sharenetworksubnets.API {
	return sharenetworksubnets.New(s.client)
}
