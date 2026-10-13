package groups

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

// GroupRecord owns the pinned Group's four nullable Body descriptors, its
// library-owned location view, and the separate actual response channels.
// Neither a response id nor an HREF supplies the next request route.
type GroupRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

func (r *GroupRecord) Clone() *GroupRecord {
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
func groupRecord(value *neutronfind.Record) *GroupRecord {
	if value == nil {
		return nil
	}
	return &GroupRecord{Resource: value.Resource, Wire: value.Wire, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode}
}
func recordInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func recordWrap(ctx context.Context, operation string, err error) error {
	return request.Wrap(operation, "groups", cloudread.ContextError(ctx, err))
}

var groupRecordProfile = neutronfind.Profile{
	ServiceRole: "identity", Kind: "groups", Path: "groups", Singular: "group", Plural: "groups", IDKey: "id", NameKey: "name",
	NameQueryHint: true, SourceFindPolicy: true, SourceStatusPolicy: true, StopFaultRetryCallbacks: true, PythonLoadedEquality: true, SourceLinkPolicy: true, VersionedPath: "/v3/groups",
	Body: []neutronfind.Descriptor{
		{Name: "description", Wire: "description", Missing: "null"},
		{Name: "domain_id", Wire: "domain_id", Missing: "null"},
		{Name: "name", Wire: "name", Missing: "null"},
		{Name: "id", Wire: "id", Missing: "null"},
	},
	Query: []neutronfind.QueryPair{{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}, {Name: "domain_id", Wire: "domain_id"}, {Name: "name", Wire: "name"}},
}

func (a *API) recordEngine(ctx context.Context) (*neutronfind.Engine, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, recordInvalid("configured groups API is required")
	}
	selected := a.client
	return neutronfind.New(ctx, selected, groupRecordProfile, func(context.Context) error {
		if a.client != selected {
			return recordInvalid("selected groups client changed")
		}
		return nil
	})
}
func applyGroupOptions[T any, O ~func(*T) error](ctx context.Context, engine *neutronfind.Engine, current T, clone func(T) T, options []O) (T, error) {
	current = clone(current)
	for _, option := range options {
		if err := engine.Check(ctx); err != nil {
			return current, err
		}
		if option == nil {
			return current, recordInvalid("nil group record option")
		}
		next := clone(current)
		if err := errors.Join(option(&next), engine.Check(ctx)); err != nil {
			return current, err
		}
		current = clone(next)
	}
	return current, engine.Check(ctx)
}
