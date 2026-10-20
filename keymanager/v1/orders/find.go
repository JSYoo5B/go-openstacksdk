package orders

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/internal/keymanagerread"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// FindIdentity implements Python find_order: a strict direct metadata Fetch,
// then on a clean 400/403/404 a complete list matched by literal id or full
// order_ref and by name, with ignore_missing true by default. Orders usually
// have no name, so list matching is in practice by reference.
func (a *API) FindIdentity(ctx context.Context, identity string, options ...resource.IdentityFindOption) (*FetchedOrder, error) {
	wrap := func(err error) error { return request.Wrap("FindIdentity", "orders", err) }
	prepared, err := resource.PrepareIdentityFindOptions(options...)
	if err != nil {
		return nil, wrap(err)
	}
	var client *gophercloud.ServiceClient
	if a != nil {
		client = a.client
	}
	record, err := keymanagerread.FindIdentity(ctx, client, "orders", identity, prepared)
	if record == nil {
		return nil, wrap(err)
	}
	value := &FetchedOrder{RequestID: record.RequestID, Resource: record.Resource, Wire: record.Wire, Envelope: record.Envelope, Header: record.Header, StatusCode: record.StatusCode}
	if record.Resource != nil {
		value.OrderID = keymanagerread.NullableString(record.Resource.Body["order_id"])
		value.SecretID = keymanagerread.NullableString(record.Resource.Body["secret_id"])
	}
	return value, wrap(err)
}
