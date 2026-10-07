// Package quotasets owns Manila's fixed-project quota API, which is absent
// from the pinned Gophercloud transport inventory.
package quotasets

import (
	"context"
	"fmt"

	"github.com/JSYoo5B/gophercloudsdk/internal/manilaversion"
	"github.com/JSYoo5B/gophercloudsdk/internal/project"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API     { return &API{client: client} }
func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }

func (a *API) validate(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if a == nil {
		return fmt.Errorf("%w: Manila quota API is required", resource.ErrInvalidOption)
	}
	if err := project.ValidateClient(ctx, a.client); err != nil {
		return err
	}
	_, err := a.minorVersion()
	return err
}

// An omitted microversion uses Manila's minimum API behavior. Nothing here
// upgrades the shared service client or guesses a version from the endpoint.
func (a *API) minorVersion() (int, error) {
	return manilaversion.Minor(a.client)
}

func (a *API) requireVersion(minor int) error {
	version, err := a.minorVersion()
	if err != nil {
		return err
	}
	if version < minor {
		return fmt.Errorf("%w: Manila operation requires microversion 2.%d", resource.ErrUnsupported, minor)
	}
	return nil
}

func (a *API) quotaURL(projectID, suffix string) (string, error) {
	minor, err := a.minorVersion()
	if err != nil {
		return "", err
	}
	root := "quota-sets"
	if minor <= 6 {
		root = "os-quota-sets"
	}
	if suffix != "" {
		return a.client.ServiceURL(root, projectID, suffix), nil
	}
	return a.client.ServiceURL(root, projectID), nil
}
