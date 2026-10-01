// Package quotasets owns Manila's fixed-project quota API, which is absent
// from the pinned Gophercloud transport inventory.
package quotasets

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/project"
	"gophercloudsdk/resource"
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
	return project.ValidateClient(ctx, a.client)
}

// An omitted microversion uses Manila's minimum API behavior. Nothing here
// upgrades the shared service client or guesses a version from the endpoint.
func (a *API) minorVersion() (int, error) {
	version := a.client.Microversion
	if version == "" {
		return 0, nil
	}
	if version == "latest" {
		return int(^uint(0) >> 1), nil
	}
	parts := strings.Split(version, ".")
	if len(parts) != 2 || parts[0] != "2" {
		return 0, fmt.Errorf("%w: Manila microversion must be 2.N or latest", resource.ErrInvalidOption)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil || minor < 0 || strconv.Itoa(minor) != parts[1] {
		return 0, fmt.Errorf("%w: invalid Manila microversion %q", resource.ErrInvalidOption, version)
	}
	return minor, nil
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
