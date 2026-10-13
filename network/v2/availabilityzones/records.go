package availabilityzones

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"maps"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
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

type AvailabilityZoneRecord = neutronfind.Record

// ListOpts mirrors the pinned proxy's **query input. That proxy discards every
// query value, so these opaque values are never marshaled, filtered or sent.
type ListOpts struct{ ProxyQuery map[string]any }
type ListOption func(*ListOpts) error

func WithListOpts(v ListOpts) ListOption {
	keys := maps.Clone(v.ProxyQuery)
	return func(c *ListOpts) error { c.ProxyQuery = maps.Clone(keys); return nil }
}
func WithProxyQuery(key string, value any) ListOption {
	return func(c *ListOpts) error {
		if c.ProxyQuery == nil {
			c.ProxyQuery = make(map[string]any)
		}
		c.ProxyQuery[key] = value
		return nil
	}
}
func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func profile() neutronfind.Profile {
	return neutronfind.Profile{Kind: "availability zones", Path: "availability_zones", VersionedPath: "/v2.0/availability_zones", Singular: "availability_zone", Plural: "availability_zones", IDKey: "id", NameKey: "name", PreserveUnknownBody: true, SourceFindPolicy: true, StopFaultRetryCallbacks: true, PythonLoadedEquality: true, Body: []neutronfind.Descriptor{{Name: "name", Wire: "name", Missing: "null"}, {Name: "resource", Wire: "resource", Missing: "null"}, {Name: "state", Wire: "state", Missing: "null"}, {Name: "id", Wire: "id", Missing: "null"}}}
}
func (a *API) engine(ctx context.Context) (*neutronfind.Engine, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, invalid("configured availability zones API required")
	}
	selected := a.client
	return neutronfind.New(ctx, selected, profile(), func(context.Context) error {
		if a.client != selected {
			return invalid("selected availability zones client changed")
		}
		return nil
	})
}

// ListRecords is the lazy availability_zones proxy contract. Its query kwargs
// are intentionally ignored, including Resource-level name/resource controls.
func (a *API) ListRecords(ctx context.Context, options ...ListOption) iter.Seq2[*AvailabilityZoneRecord, error] {
	callbacks := slices.Clone(options)
	return func(yield func(*AvailabilityZoneRecord, error) bool) {
		wrap := func(err error) error {
			return request.Wrap("ListRecords", "availability zones", cloudread.ContextError(ctx, err))
		}
		engine, err := a.engine(ctx)
		if err != nil {
			yield(nil, wrap(err))
			return
		}
		var c ListOpts
		for _, apply := range callbacks {
			if err := engine.Check(ctx); err != nil {
				yield(nil, wrap(err))
				return
			}
			if apply == nil {
				yield(nil, wrap(invalid("nil availability-zone list option")))
				return
			}
			c.ProxyQuery = maps.Clone(c.ProxyQuery)
			if err := errors.Join(apply(&c), engine.Check(ctx)); err != nil {
				yield(nil, wrap(err))
				return
			}
			c.ProxyQuery = maps.Clone(c.ProxyQuery)
		}
		if _, duplicate := c.ProxyQuery["self"]; duplicate {
			yield(nil, wrap(invalid("self collides with the bound Source proxy")))
			return
		}
		for record, err := range engine.List(ctx, neutronfind.Options{}) {
			if !yield(record, wrap(err)) {
				return
			}
		}
	}
}
func (a *API) AllRecords(ctx context.Context, options ...ListOption) ([]*AvailabilityZoneRecord, error) {
	rows := make([]*AvailabilityZoneRecord, 0)
	for row, err := range a.ListRecords(ctx, options...) {
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}
