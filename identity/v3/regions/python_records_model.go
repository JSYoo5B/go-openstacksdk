package regions

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// RegionRecord owns the pinned Region's five nullable Body descriptors, its
// library-owned location view, and the separate actual response channels.
// Neither a response id nor an HREF supplies the next request route.
type RegionRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

func (r *RegionRecord) Clone() *RegionRecord {
	if r == nil {
		return nil
	}
	c := *r
	c.Resource = r.Resource.Clone()
	c.Wire = r.Wire.Clone()
	c.Envelope = bytes.Clone(r.Envelope)
	c.Header = r.Header.Clone()
	return &c
}
func regionRecord(value *neutronfind.Record) *RegionRecord {
	if value == nil {
		return nil
	}
	return &RegionRecord{Resource: value.Resource, Wire: value.Wire, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode}
}
func recordInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func recordWrap(ctx context.Context, operation string, err error) error {
	return request.Wrap(operation, "regions", cloudread.ContextError(ctx, err))
}

var regionRecordProfile = neutronfind.Profile{
	ServiceRole: "identity", Kind: "regions", Path: "regions", Singular: "region", Plural: "regions", IDKey: "id", NameKey: "name",
	SourceFindPolicy: true, SourceStatusPolicy: true, SourceLinkPolicy: true, StopFaultRetryCallbacks: true, PythonLoadedEquality: true, VersionedPath: "/v3/regions",
	Body: []neutronfind.Descriptor{
		{Name: "description", Wire: "description", Missing: "null"},
		{Name: "links", Wire: "links", Missing: "null"},
		{Name: "parent_region_id", Wire: "parent_region_id", Missing: "null"},
		{Name: "id", Wire: "id", Missing: "null"},
		{Name: "name", Wire: "name", Missing: "null"},
	},
	Query: []neutronfind.QueryPair{{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}, {Name: "parent_region_id", Wire: "parent_region_id"}},
}

func (a *API) recordEngine(ctx context.Context) (*neutronfind.Engine, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, recordInvalid("configured regions API is required")
	}
	selected := a.client
	return neutronfind.New(ctx, selected, regionRecordProfile, func(context.Context) error {
		if a.client != selected {
			return recordInvalid("selected regions client changed")
		}
		return nil
	})
}
func applyRegionOptions[T any, O ~func(*T) error](ctx context.Context, engine *neutronfind.Engine, current T, clone func(T) T, options []O) (T, error) {
	current = clone(current)
	for _, option := range options {
		if err := engine.Check(ctx); err != nil {
			return current, err
		}
		if option == nil {
			return current, recordInvalid("nil region record option")
		}
		next := clone(current)
		if err := errors.Join(option(&next), engine.Check(ctx)); err != nil {
			return current, err
		}
		current = clone(next)
	}
	return current, engine.Check(ctx)
}
