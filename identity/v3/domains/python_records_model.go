package domains

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

// DomainRecord owns the pinned Domain's six nullable Body descriptors, its
// library-owned location view, and the separate actual response channels.
// Neither a response id nor an HREF supplies the next request route.
type DomainRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

func (r *DomainRecord) Clone() *DomainRecord {
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
func domainRecord(value *neutronfind.Record) *DomainRecord {
	if value == nil {
		return nil
	}
	return &DomainRecord{Resource: value.Resource, Wire: value.Wire, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode}
}
func recordInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func recordWrap(ctx context.Context, operation string, err error) error {
	return request.Wrap(operation, "domains", cloudread.ContextError(ctx, err))
}

var domainRecordProfile = neutronfind.Profile{
	ServiceRole: "identity", Kind: "domains", Path: "domains", Singular: "domain", Plural: "domains", IDKey: "id", NameKey: "name",
	SourceFindPolicy: true, SourceStatusPolicy: true, SourceLinkPolicy: true, StopFaultRetryCallbacks: true, PythonLoadedEquality: true, VersionedPath: "/v3/domains",
	Body: []neutronfind.Descriptor{
		{Name: "description", Wire: "description", Missing: "null"},
		{Name: "is_enabled", Wire: "enabled", Kind: neutronfind.Boolean, Missing: "null"},
		{Name: "options", Wire: "options", Kind: neutronfind.Dictionary, Missing: "null"},
		{Name: "links", Wire: "links", Missing: "null"},
		{Name: "id", Wire: "id", Missing: "null"},
		{Name: "name", Wire: "name", Missing: "null"},
	},
	Query: []neutronfind.QueryPair{{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}, {Name: "name", Wire: "name"}, {Name: "is_enabled", Wire: "enabled"}},
}

func (a *API) recordEngine(ctx context.Context) (*neutronfind.Engine, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, recordInvalid("configured domains API is required")
	}
	selected := a.client
	return neutronfind.New(ctx, selected, domainRecordProfile, func(context.Context) error {
		if a.client != selected {
			return recordInvalid("selected domains client changed")
		}
		return nil
	})
}
func applyDomainOptions[T any, O ~func(*T) error](ctx context.Context, engine *neutronfind.Engine, current T, clone func(T) T, options []O) (T, error) {
	current = clone(current)
	for _, option := range options {
		if err := engine.Check(ctx); err != nil {
			return current, err
		}
		if option == nil {
			return current, recordInvalid("nil domain record option")
		}
		next := clone(current)
		if err := errors.Join(option(&next), engine.Check(ctx)); err != nil {
			return current, err
		}
		current = clone(next)
	}
	return current, engine.Check(ctx)
}
