package v1

import (
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/chassis"
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/portgroups"
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/volumeconnectors"
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/volumetargets"
)

// Chassis returns the SDK-owned /v1/chassis binding on this service's client.
// Each call creates a stateless API value; no HTTP is performed.
func (s *Service) Chassis() *chassis.API { return chassis.New(s.client) }

// PortGroups returns the SDK-owned /v1/portgroups binding.
func (s *Service) PortGroups() *portgroups.API { return portgroups.New(s.client) }

// VolumeConnectors returns the SDK-owned /v1/volume/connectors binding.
func (s *Service) VolumeConnectors() *volumeconnectors.API { return volumeconnectors.New(s.client) }

// VolumeTargets returns the SDK-owned /v1/volume/targets binding.
func (s *Service) VolumeTargets() *volumetargets.API { return volumetargets.New(s.client) }
