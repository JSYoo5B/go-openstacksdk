// Package objectstorage owns high-level Swift cloud workflows.
package objectstorage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Cloud shares its authenticated client with the normal Swift resource APIs.
type Cloud struct {
	client     *gophercloud.ServiceClient
	containers *containers.API
}

func New(client *gophercloud.ServiceClient) *Cloud {
	return &Cloud{client: client, containers: containers.New(client)}
}
func (c *Cloud) RawClient() *gophercloud.ServiceClient {
	if c == nil {
		return nil
	}
	return c.client
}

// ContainerCreateResult keeps the initial observation, PUT acknowledgement and
// final HEAD separate. Container is the Source return value, including nil if
// the container disappears after creation; a later failure never replays PUT.
type ContainerCreateResult struct {
	Container *containers.ContainerRecord
	Initial   *containers.ContainerRecord
	Creation  *containers.ContainerRecord
	Refreshed *containers.ContainerRecord
	Existing  bool
}

// ContainerAccessUpdateResult contains actual POST and optional HEAD records.
// Requested is an input, not proof of the container's current ACL.
type ContainerAccessUpdateResult struct {
	Requested       string
	Acknowledgement *containers.ContainerRecord
	Refreshed       *containers.ContainerRecord
	Container       *containers.ContainerRecord
}

// ContainerAccessResult carries the classified access and the HEAD evidence.
type ContainerAccessResult struct {
	Access    string
	Container *containers.ContainerRecord
}

func (c *Cloud) prepareContainer(ctx context.Context, name string) (*containers.CloudRecords, context.Context, error) {
	if ctx == nil {
		return nil, ctx, fmt.Errorf("%w: context is required", resource.ErrInvalidOption)
	}
	if c == nil || c.containers == nil {
		return nil, ctx, fmt.Errorf("%w: object-store cloud is required", resource.ErrInvalidOption)
	}
	ctx = rest.WithOperationSources(ctx)
	prepared, err := c.containers.PrepareCloudRecords(ctx, name)
	return prepared, ctx, err
}
func containerCloudError(ctx context.Context, operation string, err error) error {
	if ctx != nil && ctx.Err() != nil {
		err = errors.Join(err, ctx.Err(), context.Cause(ctx))
	}
	return request.Wrap(operation, "containers", err)
}
func containerCloudMissing(err error) bool { return err != nil && resource.IgnoreMissing(err) == nil }

// CreateContainer returns an existing container, or creates and re-fetches it.
// The default public=false emits no operation-owned ACL on PUT. Existing ACLs
// are preserved even when WithContainerPublic(true) is supplied.
func (c *Cloud) CreateContainer(ctx context.Context, name string, options ...ContainerCreateOption) (*ContainerCreateResult, error) {
	const operation = "CreateContainer"
	prepared, scoped, err := c.prepareContainer(ctx, name)
	if err != nil {
		return nil, containerCloudError(ctx, operation, err)
	}
	cfg, err := applyContainerCreateOptions(scoped, prepared, options)
	if err != nil {
		return nil, containerCloudError(scoped, operation, err)
	}
	if err = prepared.SetHeaders(scoped, cfg.Headers); err != nil {
		return nil, containerCloudError(scoped, operation, err)
	}
	result := &ContainerCreateResult{}
	result.Initial, err = prepared.Get(scoped)
	if err == nil {
		result.Container, result.Existing = result.Initial, true
		return result, nil
	}
	if !containerCloudMissing(err) {
		return result, containerCloudError(scoped, operation, err)
	}
	result.Creation, err = prepared.Create(scoped, cfg.Public)
	if err != nil {
		return result, containerCloudError(scoped, operation, err)
	}
	result.Refreshed, err = prepared.Get(scoped)
	if containerCloudMissing(err) {
		return result, nil
	}
	if err == nil {
		result.Container = result.Refreshed
	}
	return result, containerCloudError(scoped, operation, err)
}

// SetContainerAccess accepts only public and private. The cloud default refresh
// is false; WithContainerAccessRefresh(true) observes state with a second HEAD.
func (c *Cloud) SetContainerAccess(ctx context.Context, name, access string, options ...ContainerAccessOption) (*ContainerAccessUpdateResult, error) {
	const operation = "SetContainerAccess"
	acl, ok := map[string]string{"public": ".r:*,.rlistings", "private": ""}[access]
	if !ok {
		return nil, containerCloudError(ctx, operation, fmt.Errorf("%w: invalid container access %q; expected public or private", resource.ErrInvalidOption, access))
	}
	prepared, scoped, err := c.prepareContainer(ctx, name)
	if err != nil {
		return nil, containerCloudError(ctx, operation, err)
	}
	cfg, err := applyContainerAccessOptions(scoped, prepared, options)
	if err != nil {
		return nil, containerCloudError(scoped, operation, err)
	}
	if err = prepared.SetHeaders(scoped, cfg.Headers); err != nil {
		return nil, containerCloudError(scoped, operation, err)
	}
	result := &ContainerAccessUpdateResult{Requested: access}
	result.Acknowledgement, err = prepared.SetAccess(scoped, acl)
	if err != nil {
		return result, containerCloudError(scoped, operation, err)
	}
	result.Container = result.Acknowledgement
	if !cfg.Refresh {
		return result, nil
	}
	result.Refreshed, err = prepared.Get(scoped)
	if err == nil {
		result.Container = result.Refreshed
	} else {
		result.Container = nil
	}
	return result, containerCloudError(scoped, operation, err)
}

// GetContainerAccess classifies the exact observed ACL. A missing/empty ACL is
// private; unknown ACLs and missing containers are errors with HEAD evidence.
func (c *Cloud) GetContainerAccess(ctx context.Context, name string, options ...ContainerReadOption) (*ContainerAccessResult, error) {
	const operation = "GetContainerAccess"
	prepared, scoped, err := c.prepareContainer(ctx, name)
	if err != nil {
		return nil, containerCloudError(ctx, operation, err)
	}
	cfg, err := applyContainerReadOptions(scoped, prepared, options)
	if err != nil {
		return nil, containerCloudError(scoped, operation, err)
	}
	if err = prepared.SetHeaders(scoped, cfg.Headers); err != nil {
		return nil, containerCloudError(scoped, operation, err)
	}
	result := &ContainerAccessResult{}
	result.Container, err = prepared.Get(scoped)
	if containerCloudMissing(err) {
		return result, containerCloudError(scoped, operation, &resource.NotFoundError{Resource: "container", Reference: name, Cause: err})
	}
	if err != nil {
		return result, containerCloudError(scoped, operation, err)
	}
	raw := result.Container.Resource.Body["read_ACL"]
	truthy, err := cloudfilter.PythonJSONTruthy(raw)
	if err != nil {
		return result, containerCloudError(scoped, operation, err)
	}
	if !truthy {
		result.Access = "private"
		return result, nil
	}
	var acl string
	if err = json.Unmarshal(raw, &acl); err != nil {
		return result, containerCloudError(scoped, operation, err)
	}
	switch acl {
	case ".r:*,.rlistings":
		result.Access = "public"
	case "":
		result.Access = "private"
	default:
		return result, containerCloudError(scoped, operation, fmt.Errorf("could not determine container access for ACL %q", acl))
	}
	return result, nil
}
