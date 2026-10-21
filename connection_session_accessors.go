package openstack

import (
	"context"
	"errors"
	"fmt"

	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
)

// ErrNoAuthResult reports that the Connection has no recorded Keystone token
// response, for example because it adopted a provider whose token was set by
// hand. CurrentUserID and ServiceCatalog cannot answer in that case.
var ErrNoAuthResult = errors.New("connection has no recorded Keystone authentication result")

// CatalogEndpoint is one endpoint of a token service catalog entry.
type CatalogEndpoint struct {
	ID        string
	Interface string
	Region    string
	RegionID  string
	URL       string
}

// CatalogService is one service entry of the token service catalog.
type CatalogService struct {
	ID        string
	Name      string
	Type      string
	Endpoints []CatalogEndpoint
}

// AuthToken returns the token currently held by the shared provider without any
// HTTP request. It is empty when no token has been obtained, and unlike the
// keystoneauth accessor it never re-authenticates an expired token.
func (c *Connection) AuthToken() string {
	return c.provider.Token()
}

// Authorize returns the current token, authenticating through the provider's
// reauthentication function first when no token is held. It does not refresh a
// token that is merely expired; the failed request triggers native reauth.
// Errors wrap the native cause and context cancellation stays observable.
func (c *Connection) Authorize(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("authorize: %w", err)
	}
	if token := c.provider.Token(); token != "" {
		return token, nil
	}
	if err := c.provider.Reauthenticate(ctx, ""); err != nil {
		return "", fmt.Errorf("authorize: %w", err)
	}
	token := c.provider.Token()
	if token == "" {
		return "", invalid("authorize: provider has no token and no reauthentication function")
	}
	return token, nil
}

// CurrentUserID returns the user ID recorded in the Keystone token response.
// It makes no request and returns ErrNoAuthResult when no response is recorded.
func (c *Connection) CurrentUserID() (string, error) {
	r, ok := c.provider.GetAuthResult().(interface {
		ExtractUser() (*tokens.User, error)
	})
	if !ok {
		return "", ErrNoAuthResult
	}
	user, err := r.ExtractUser()
	if err != nil {
		return "", fmt.Errorf("decode token user: %w", err)
	}
	if user == nil {
		return "", nil
	}
	return user.ID, nil
}

// ServiceCatalog returns a copy of the service catalog recorded in the Keystone
// token response. It makes no request and returns ErrNoAuthResult when no
// response is recorded. An absent catalog yields an empty slice.
func (c *Connection) ServiceCatalog() ([]CatalogService, error) {
	r, ok := c.provider.GetAuthResult().(interface {
		ExtractServiceCatalog() (*tokens.ServiceCatalog, error)
	})
	if !ok {
		return nil, ErrNoAuthResult
	}
	catalog, err := r.ExtractServiceCatalog()
	if err != nil {
		return nil, fmt.Errorf("decode token service catalog: %w", err)
	}
	out := make([]CatalogService, 0, len(catalog.Entries))
	for _, entry := range catalog.Entries {
		service := CatalogService{ID: entry.ID, Name: entry.Name, Type: entry.Type, Endpoints: make([]CatalogEndpoint, 0, len(entry.Endpoints))}
		for _, e := range entry.Endpoints {
			service.Endpoints = append(service.Endpoints, CatalogEndpoint{ID: e.ID, Interface: e.Interface, Region: e.Region, RegionID: e.RegionID, URL: e.URL})
		}
		out = append(out, service)
	}
	return out, nil
}
