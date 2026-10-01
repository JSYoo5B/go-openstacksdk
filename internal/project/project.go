// Package project owns project reference and recorded authentication resolution
// for SDK services. Applications use their Connection or service scope helpers.
package project

import (
	"context"
	"fmt"
	"reflect"

	"github.com/gophercloud/gophercloud/v2"
	tokens2 "github.com/gophercloud/gophercloud/v2/openstack/identity/v2/tokens"
	tokens3 "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
	"gophercloudsdk/identity/v3/projects"
	"gophercloudsdk/resource"
)

func ValidateClient(ctx context.Context, client *gophercloud.ServiceClient) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if client == nil || client.ProviderClient == nil {
		return fmt.Errorf("%w: project scopes require a service client", resource.ErrInvalidOption)
	}
	return nil
}

func ValidateIdentityClient(client *gophercloud.ServiceClient) error {
	if client == nil || client.ProviderClient == nil || client.Type != "identity" {
		return fmt.Errorf("%w: project names require an Identity v3 service client", resource.ErrInvalidOption)
	}
	return nil
}

// Resolve makes explicit IDs request-free and exact names use a separate
// Keystone project collection. The returned ID is validated before use.
func Resolve(ctx context.Context, service *gophercloud.ServiceClient, ref resource.Ref, identity *gophercloud.ServiceClient) (string, error) {
	if err := ValidateClient(ctx, service); err != nil {
		return "", err
	}
	if err := ref.Validate(); err != nil {
		return "", err
	}
	if !ref.IsName() {
		return ref.String(), nil
	}
	if identity == nil {
		return "", fmt.Errorf("%w: project name resolution needs an Identity v3 client", resource.ErrUnsupported)
	}
	if err := ValidateIdentityClient(identity); err != nil {
		return "", err
	}
	if identity == service {
		return "", fmt.Errorf("%w: a service client cannot resolve its own project names", resource.ErrInvalidOption)
	}
	return projects.New(identity).Resources.ResolveID(ctx, ref)
}

// Current reads the native Keystone result, never guessing an ID from endpoint
// paths or token strings. It does not refresh authentication or cache the ID.
func Current(ctx context.Context, client *gophercloud.ServiceClient) (string, error) {
	if err := ValidateClient(ctx, client); err != nil {
		return "", err
	}
	auth := client.ProviderClient.GetAuthResult()
	if auth == nil || reflect.ValueOf(auth).Kind() == reflect.Pointer && reflect.ValueOf(auth).IsNil() {
		return "", fmt.Errorf("%w: no project authentication result; supply an explicit project ID", resource.ErrUnsupported)
	}
	var id string
	switch result := auth.(type) {
	case interface {
		ExtractProject() (*tokens3.Project, error)
	}:
		project, err := result.ExtractProject()
		if err != nil {
			return "", err
		}
		if project != nil {
			id = project.ID
		}
	case interface {
		ExtractToken() (*tokens2.Token, error)
	}:
		token, err := result.ExtractToken()
		if err != nil {
			return "", err
		}
		if token != nil {
			id = token.Tenant.ID
		}
	default:
		return "", fmt.Errorf("%w: authentication result %T does not expose a Keystone project", resource.ErrUnsupported, auth)
	}
	if id == "" {
		return "", fmt.Errorf("%w: authentication is not project-scoped; supply an explicit project ID", resource.ErrUnsupported)
	}
	if err := resource.ID(id).Validate(); err != nil {
		return "", err
	}
	return id, ctx.Err()
}
