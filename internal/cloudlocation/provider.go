// Package cloudlocation reads already recorded native token scope without I/O.
package cloudlocation

import (
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	v2 "github.com/gophercloud/gophercloud/v2/openstack/identity/v2/tokens"
	v3 "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
)

// ProjectID uses only known native authentication result shapes. Manually set
// tokens and custom results without recorded scope yield null, not a guessed
// configured project or a new authentication request.
func ProjectID(provider *gophercloud.ProviderClient) (json.RawMessage, error) {
	if provider == nil {
		return nil, fmt.Errorf("%w: provider is required", resource.ErrInvalidOption)
	}
	result := provider.GetAuthResult()
	var id string
	var err error
	switch value := result.(type) {
	case v3.CreateResult:
		id, err = projectV3(value.ExtractProject)
	case *v3.CreateResult:
		if value != nil {
			id, err = projectV3(value.ExtractProject)
		}
	case v3.GetResult:
		id, err = projectV3(value.ExtractProject)
	case *v3.GetResult:
		if value != nil {
			id, err = projectV3(value.ExtractProject)
		}
	case v2.CreateResult:
		id, err = projectV2(value)
	case *v2.CreateResult:
		if value != nil {
			id, err = projectV2(*value)
		}
	case v2.GetResult:
		id, err = projectV2(value.CreateResult)
	case *v2.GetResult:
		if value != nil {
			id, err = projectV2(value.CreateResult)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("%w: recorded authentication project: %w", resource.ErrInvalidOption, err)
	}
	if id == "" {
		return nil, nil
	}
	return json.Marshal(id)
}

func projectV3(extract func() (*v3.Project, error)) (string, error) {
	project, err := extract()
	if err != nil || project == nil {
		return "", err
	}
	return project.ID, nil
}

func projectV2(result v2.CreateResult) (string, error) {
	// ExtractToken also parses expiry; location needs only the recorded scope.
	var body struct {
		Access struct {
			Token struct {
				Tenant struct {
					ID string `json:"id"`
				} `json:"tenant"`
			} `json:"token"`
		} `json:"access"`
	}
	err := result.ExtractInto(&body)
	return body.Access.Token.Tenant.ID, err
}
