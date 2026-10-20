// Package capabilities reads Cinder backend capabilities with owned Resource views.
package capabilities

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }

func (a *API) RawClient() *gophercloud.ServiceClient {
	if a == nil {
		return nil
	}
	return a.client
}

type Record struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}
type ReadOpts struct{ Headers map[string]string }
type ReadOption func(*ReadOpts) error

func WithHeaders(v map[string]string) ReadOption {
	owned := maps.Clone(v)
	return func(o *ReadOpts) error { o.Headers = maps.Clone(owned); return nil }
}
func (a *API) GetRecord(ctx context.Context, host string, options ...ReadOption) (*Record, error) {
	wrap := func(err error) error { return request.Wrap("GetRecord", "capabilities", err) }
	if a == nil {
		return nil, wrap(resource.ErrInvalidOption)
	}
	profile := neutronfind.Profile{ServiceRole: "volume", Kind: "capabilities", Path: "capabilities", IDKey: "id", NameKey: "name", SourceFindPolicy: true, SourceStatusPolicy: true, StopFaultRetryCallbacks: true}
	for _, name := range []string{"id", "name", "description", "display_name", "driver_version", "namespace", "pool_name", "properties", "replication_targets", "storage_protocol", "vendor_name", "visibility", "volume_backend_name"} {
		field := neutronfind.Descriptor{Name: name, Wire: name, Missing: "null"}
		if name == "properties" {
			field.Kind = neutronfind.Dictionary
		}
		if name == "replication_targets" {
			field.Kind = neutronfind.List
		}
		profile.Body = append(profile.Body, field)
	}
	selected := a.client
	engine, err := neutronfind.New(ctx, selected, profile, func(context.Context) error {
		if a.client != selected {
			return resource.ErrInvalidOption
		}
		return nil
	})
	if err != nil {
		return nil, wrap(err)
	}
	policy := ReadOpts{}
	for _, apply := range options {
		if apply == nil {
			return nil, wrap(resource.ErrInvalidOption)
		}
		next := ReadOpts{Headers: maps.Clone(policy.Headers)}
		if err := engine.Check(ctx); err != nil {
			return nil, wrap(err)
		}
		err = errors.Join(apply(&next), engine.Check(ctx))
		if err != nil {
			return nil, wrap(err)
		}
		policy = ReadOpts{Headers: maps.Clone(next.Headers)}
	}
	value, err := engine.Get(ctx, host, neutronfind.Options{Headers: policy.Headers})
	if value == nil {
		return nil, wrap(err)
	}
	return &Record{Resource: value.Resource, Wire: value.Wire, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode}, wrap(err)
}
