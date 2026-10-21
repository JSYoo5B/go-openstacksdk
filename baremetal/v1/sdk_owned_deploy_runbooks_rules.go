package v1

import (
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/deploytemplates"
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/inspectionrules"
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/runbooks"
)

// DeployTemplatesAPI returns the SDK-owned deploy template API on the same
// authenticated client as the native resources.
func (s *Service) DeployTemplatesAPI() *deploytemplates.API { return deploytemplates.New(s.client) }

// InspectionRulesAPI returns the SDK-owned Ironic inspection rule API.
func (s *Service) InspectionRulesAPI() *inspectionrules.API { return inspectionrules.New(s.client) }

// RunbooksAPI returns the SDK-owned runbook API, including runbook traits.
func (s *Service) RunbooksAPI() *runbooks.API { return runbooks.New(s.client) }
