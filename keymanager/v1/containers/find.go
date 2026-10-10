package containers

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/internal/keymanagerread"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// FindIdentity implements Python find_container: a strict direct metadata
// Fetch, then on a clean 400/403/404 a complete list matched by literal id or
// full container_ref and by name, with ignore_missing true by default. List
// rows are projected without a request seed and never trigger another GET.
func (a *API) FindIdentity(ctx context.Context, identity string, options ...resource.IdentityFindOption) (*FetchedContainer, error) {
	wrap := func(err error) error { return request.Wrap("FindIdentity", "containers", err) }
	prepared, err := resource.PrepareIdentityFindOptions(options...)
	if err != nil {
		return nil, wrap(err)
	}
	var client *gophercloud.ServiceClient
	if a != nil {
		client = a.client
	}
	record, err := keymanagerread.FindIdentity(ctx, client, "containers", identity, prepared)
	if record == nil {
		return nil, wrap(err)
	}
	value := &FetchedContainer{RequestID: record.RequestID, Resource: record.Resource, Wire: record.Wire, Envelope: record.Envelope, Header: record.Header, StatusCode: record.StatusCode}
	if record.Resource != nil {
		value.ContainerID = keymanagerread.NullableString(record.Resource.Body["container_id"])
	}
	return value, wrap(err)
}
