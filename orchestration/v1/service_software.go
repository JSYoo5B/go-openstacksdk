package v1

import (
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/softwareconfigs"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/softwaredeployments"
)

// SoftwareConfigs returns the SDK-owned /software_configs API on the shared
// authenticated client. Gophercloud v2.15.0 has no binding for this route.
func (s *Service) SoftwareConfigs() *softwareconfigs.API { return softwareconfigs.New(s.client) }

// SoftwareDeployments returns the SDK-owned /software_deployments API on the
// shared authenticated client.
func (s *Service) SoftwareDeployments() *softwaredeployments.API {
	return softwaredeployments.New(s.client)
}
