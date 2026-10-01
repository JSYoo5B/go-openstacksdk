package limits

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/project"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

type projectOptions struct{ identity *gophercloud.ServiceClient }
type ProjectOption func(*projectOptions) error

func WithIdentityClient(client *gophercloud.ServiceClient) ProjectOption {
	return func(config *projectOptions) error {
		if err := project.ValidateIdentityClient(client); err != nil {
			return err
		}
		config.identity = client
		return nil
	}
}

type ProjectLimitsScope struct {
	api       *API
	projectID string
}

func (a *API) validateClient(ctx context.Context) error {
	var client *gophercloud.ServiceClient
	if a != nil {
		client = a.client
	}
	if err := project.ValidateClient(ctx, client); err != nil {
		return err
	}
	_, err := a.minorVersion()
	return err
}

// minorVersion validates the version that the request will actually send.
// Native service MoreHeaders override automatically generated version headers.
func (a *API) minorVersion() (int, error) {
	selected := a.client.Microversion
	expected := selected
	if expected == "" {
		expected = "3.0"
	}
	minor := int(^uint(0) >> 1)
	if expected != "latest" {
		parts := strings.Split(expected, ".")
		if len(parts) != 2 || parts[0] != "3" {
			return 0, fmt.Errorf("%w: Cinder microversion must be 3.N or latest", resource.ErrInvalidOption)
		}
		var err error
		minor, err = strconv.Atoi(parts[1])
		if err != nil || minor < 0 || strconv.Itoa(minor) != parts[1] {
			return 0, fmt.Errorf("%w: invalid Cinder microversion %q", resource.ErrInvalidOption, selected)
		}
	}
	explicitVersion := false
	for key, value := range a.client.MoreHeaders {
		switch {
		case strings.EqualFold(key, "X-OpenStack-Volume-API-Version"):
			if value != expected {
				return 0, fmt.Errorf("%w: Cinder version header conflicts with selected microversion", resource.ErrInvalidOption)
			}
			explicitVersion = true
		case strings.EqualFold(key, "OpenStack-API-Version"):
			if value != "volume "+expected {
				return 0, fmt.Errorf("%w: Cinder version header conflicts with selected microversion", resource.ErrInvalidOption)
			}
			explicitVersion = true
		}
	}
	if selected != "" {
		switch a.client.Type {
		case "block-storage", "block-store", "volume", "volumev3":
		case "":
			if !explicitVersion {
				return 0, fmt.Errorf("%w: Cinder microversion requires a Cinder client type or explicit matching version header", resource.ErrInvalidOption)
			}
		default:
			return 0, fmt.Errorf("%w: Cinder limits require a Cinder service client", resource.ErrInvalidOption)
		}
	}
	return minor, nil
}

func (a *API) requireProjectFilter() error {
	minor, err := a.minorVersion()
	if err != nil {
		return err
	}
	if minor < 39 {
		return fmt.Errorf("%w: Cinder project limits filtering requires microversion 3.39", resource.ErrUnsupported)
	}
	return nil
}

// InProject checks 3.39 before any exact-name lookup, then fixes the target ID.
// No endpoint component or authentication token is interpreted as a project.
func (a *API) InProject(ctx context.Context, ref resource.Ref, options ...ProjectOption) (*ProjectLimitsScope, error) {
	if err := a.validateClient(ctx); err != nil {
		return nil, limitsError("InProject", ref.String(), err)
	}
	if err := a.requireProjectFilter(); err != nil {
		return nil, limitsError("InProject", ref.String(), err)
	}
	var config projectOptions
	for _, apply := range options {
		if apply == nil {
			return nil, limitsError("InProject", ref.String(), fmt.Errorf("%w: nil project option", resource.ErrInvalidOption))
		}
		if err := apply(&config); err != nil {
			return nil, limitsError("InProject", ref.String(), err)
		}
	}
	id, err := project.Resolve(ctx, a.client, ref, config.identity)
	if err != nil {
		return nil, limitsError("InProject", ref.String(), err)
	}
	return &ProjectLimitsScope{api: a, projectID: id}, nil
}

func (a *API) CurrentProject(ctx context.Context) (*ProjectLimitsScope, error) {
	if err := a.validateClient(ctx); err != nil {
		return nil, limitsError("CurrentProject", "", err)
	}
	if err := a.requireProjectFilter(); err != nil {
		return nil, limitsError("CurrentProject", "", err)
	}
	id, err := project.Current(ctx, a.client)
	if err != nil {
		return nil, limitsError("CurrentProject", "", err)
	}
	return &ProjectLimitsScope{api: a, projectID: id}, nil
}

func (s *ProjectLimitsScope) ProjectID() string {
	if s == nil {
		return ""
	}
	return s.projectID
}

func (s *ProjectLimitsScope) validate(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil {
		return fmt.Errorf("%w: project limits scope is required", resource.ErrInvalidOption)
	}
	if err := s.api.validateClient(ctx); err != nil {
		return err
	}
	if err := s.api.requireProjectFilter(); err != nil {
		return err
	}
	return resource.ID(s.projectID).Validate()
}

func limitsError(operation, projectID string, err error) error {
	if gophercloud.ResponseCodeIs(err, 404) {
		err = &resource.NotFoundError{Resource: "block storage limits", Reference: projectID, Cause: err}
	}
	return request.Wrap(operation, "block storage limits", err)
}
