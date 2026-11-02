package grouptypes

import (
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/internal/cinderowned"
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// GetOpts carries per-request headers, a Go extension. It is shared by Get
// and the group_specs calls other than DeleteGroupSpecsProperty.
type GetOpts = cinderowned.ReadOpts

// GetOption is one option of Get, FetchGroupSpecs, CreateGroupSpecs,
// GetGroupSpecsProperty and UpdateGroupSpecsProperty: request.Headers
// (replaces the headers) or request.Custom over *GetOpts. Other shared calls
// fail with resource.ErrInvalidOption before HTTP.
type GetOption = request.GetCall

// CreateOpts carries the Python **attrs as raw JSON. Keys are the declared
// attributes id, name, description, group_specs and is_public; unknown keys
// are dropped.
type CreateOpts = cinderowned.WriteOpts

// CreateOption is one option of Create: WithCreateOpts, request.Field (a
// declared attribute; unknown keys are dropped), request.Headers (replaces
// the headers) or request.Custom over *CreateOpts. Other shared calls fail
// with resource.ErrInvalidOption before HTTP.
type CreateOption = request.WriteCall

func WithCreateOpts(opts CreateOpts) CreateOption {
	return request.Custom(cinderowned.WriteSnapshot(opts))
}

// UpdateOpts carries the Python **attrs of update_group_type; id is never
// sent.
type UpdateOpts = cinderowned.WriteOpts

// UpdateOption is one option of Update: WithUpdateOpts, request.Field (a
// declared attribute; unknown keys are dropped), request.Headers (replaces
// the headers) or request.Custom over *UpdateOpts. Other shared calls fail
// with resource.ErrInvalidOption before HTTP.
type UpdateOption = request.WriteCall

func WithUpdateOpts(opts UpdateOpts) UpdateOption {
	return request.Custom(cinderowned.WriteSnapshot(opts))
}

// DeleteOpts selects ignore_missing; nil keeps the Python default true.
type DeleteOpts = cinderowned.DeleteOpts

// DeleteOption is one option of Delete and DeleteGroupSpecsProperty:
// request.IgnoreMissing, request.Headers (replaces the headers) or
// request.Custom over *DeleteOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type DeleteOption = request.DeleteCall

// FindOpts selects ignore_missing; nil keeps the Python default true.
type FindOpts = cinderowned.FindOpts

// FindOption is one option of Find: request.IgnoreMissing or request.Custom
// over *FindOpts. Other shared calls fail with resource.ErrInvalidOption
// before HTTP.
type FindOption = request.FindCall

// ListOpts carries Python **query: is_public, offset, sort_dir, sort_key,
// sort and all_projects (sent as all_tenants) go to the server; id, name,
// description and group_specs filter locally; other names are ignored.
type ListOpts = cinderowned.ListOpts

// ListOption is one option of List and All: WithListOpts, request.Query (a
// Python **query name; the value is sent as JSON), request.Limit,
// request.Marker, request.MaxItems, request.Paginated, request.Headers
// (replaces the headers) or request.Custom over *ListOpts. Other shared calls
// fail with resource.ErrInvalidOption before HTTP.
type ListOption = request.ListCall

func WithListOpts(opts ListOpts) ListOption { return request.Custom(cinderowned.ListSnapshot(opts)) }

var (
	getCalls    = calls.Spec[GetOpts]{Headers: cinderowned.ReadHeadersCall}
	writeCalls  = calls.Spec[cinderowned.WriteOpts]{Headers: cinderowned.WriteHeadersCall, Field: spec.WriteFieldCall}
	deleteCalls = calls.Spec[DeleteOpts]{Headers: cinderowned.DeleteHeadersCall, IgnoreMissing: cinderowned.DeleteIgnoreMissingCall}
	findCalls   = calls.Spec[FindOpts]{IgnoreMissing: cinderowned.FindIgnoreMissingCall}
	listCalls   = calls.Spec[ListOpts]{
		Headers: cinderowned.ListHeadersCall(cinderowned.Self[ListOpts]),
		Query:   cinderowned.ListQueryCall(spec, cinderowned.Self[ListOpts]),
		Paging:  cinderowned.ListPaging(cinderowned.Self[ListOpts]),
	}
)
