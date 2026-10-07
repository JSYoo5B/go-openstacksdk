package orders

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/keymanagerread"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// FetchOpts has no default query/name/wait policy. Headers are concrete options.
type FetchOpts struct{}
type FetchOption = request.Option[FetchOpts]

func WithFetchHeader(key, value string) FetchOption { return request.WithHeader[FetchOpts](key, value) }

// FetchedOrder separates fixed request identity, normalized attributes and
// actual nullable/extension fields. Passive references never redirect requests.
type FetchedOrder struct {
	RequestID         string
	Resource, Wire    *resource.RawResource
	Envelope          json.RawMessage
	Header            http.Header
	StatusCode        int
	OrderID, SecretID *string
}

// Ref maps a fetched Go value to its original executable request ID. It does
// not route from response id or the response's full resource reference.
func (value *FetchedOrder) Ref() resource.Ref {
	if value == nil {
		return resource.Ref{}
	}
	return resource.ID(value.RequestID)
}

// Fetch is the owned mapping of Python get_order; generated Get
// keeps its existing native alias and return type for compatibility.
func (a *API) Fetch(ctx context.Context, ref resource.Ref, options ...FetchOption) (*FetchedOrder, error) {
	var client *gophercloud.ServiceClient
	if a != nil {
		client = a.client
	}
	record, err := keymanagerread.Fetch(ctx, client, "orders", ref, options...)
	if record == nil {
		return nil, request.Wrap("Fetch", "orders", err)
	}
	value := &FetchedOrder{RequestID: record.RequestID, Resource: record.Resource, Wire: record.Wire, Envelope: record.Envelope, Header: record.Header, StatusCode: record.StatusCode}
	if record.Resource != nil {
		value.OrderID = keymanagerread.NullableString(record.Resource.Body["order_id"])
		value.SecretID = keymanagerread.NullableString(record.Resource.Body["secret_id"])
	}
	return value, request.Wrap("Fetch", "orders", err)
}
