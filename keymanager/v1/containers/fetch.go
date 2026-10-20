package containers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/keymanagerread"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// FetchOpts has no default query/name/wait policy. Headers are concrete options.
type FetchOpts struct{}
type FetchOption = request.Option[FetchOpts]

func WithFetchHeader(key, value string) FetchOption { return request.WithHeader[FetchOpts](key, value) }

// FetchedContainer separates fixed request identity, normalized attributes and
// actual nullable/extension fields. Passive references never redirect requests.
type FetchedContainer struct {
	RequestID      string
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	ContainerID    *string
}

// Ref maps a fetched Go value to its original executable request ID. It does
// not route from response id or the response's full resource reference.
func (value *FetchedContainer) Ref() resource.Ref {
	if value == nil {
		return resource.Ref{}
	}
	return resource.ID(value.RequestID)
}

// Fetch is the owned mapping of Python get_container; generated Get
// keeps its existing native alias and return type for compatibility.
func (a *API) Fetch(ctx context.Context, ref resource.Ref, options ...FetchOption) (*FetchedContainer, error) {
	var client *gophercloud.ServiceClient
	if a != nil {
		client = a.client
	}
	record, err := keymanagerread.Fetch(ctx, client, "containers", ref, options...)
	if record == nil {
		return nil, request.Wrap("Fetch", "containers", err)
	}
	value := &FetchedContainer{RequestID: record.RequestID, Resource: record.Resource, Wire: record.Wire, Envelope: record.Envelope, Header: record.Header, StatusCode: record.StatusCode}
	if record.Resource != nil {
		value.ContainerID = keymanagerread.NullableString(record.Resource.Body["container_id"])
	}
	return value, request.Wrap("Fetch", "containers", err)
}
