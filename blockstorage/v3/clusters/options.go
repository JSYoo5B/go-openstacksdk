package clusters

import (
	"maps"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/internal/cinderowned"
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// GetOpts carries per-request headers, a Go extension.
type GetOpts = cinderowned.ReadOpts

// GetOption is one option of Get: request.Headers (replaces the headers) or
// request.Custom over *GetOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type GetOption = request.GetCall

// FindOpts selects ignore_missing; nil keeps the Python default true.
type FindOpts = cinderowned.FindOpts

// FindOption is one option of Find: request.IgnoreMissing or request.Custom
// over *FindOpts. Other shared calls fail with resource.ErrInvalidOption
// before HTTP.
type FindOption = request.FindCall

// ListOpts carries Python details (nil means true) and **query: name,
// binary, is_up, disabled, num_hosts, num_down_hosts, replication_status,
// frozen and active_backend_id go to the server; id, created_at,
// disabled_reason, last_heartbeat, state, status and updated_at filter
// locally; other names are ignored.
type ListOpts struct {
	cinderowned.ListOpts
	Details *bool
}

// ListOption is one option of List and All: WithListOpts, WithListDetails,
// request.Query (a Python **query name; the value is sent as JSON),
// request.Limit, request.Marker, request.MaxItems, request.Paginated,
// request.Headers (replaces the headers) or request.Custom over *ListOpts.
// Other shared calls fail with resource.ErrInvalidOption before HTTP.
type ListOption = request.ListCall

func copyList(v ListOpts) ListOpts {
	v.ListOpts = cinderowned.CopyList(v.ListOpts)
	if v.Details != nil {
		x := *v.Details
		v.Details = &x
	}
	return v
}

func inner(v *ListOpts) *cinderowned.ListOpts { return &v.ListOpts }

func WithListOpts(opts ListOpts) ListOption {
	owned := copyList(opts)
	return request.Custom(func(v *ListOpts) error { *v = copyList(owned); return nil })
}
func WithListDetails(value bool) ListOption {
	return request.Custom(func(v *ListOpts) error { x := value; v.Details = &x; return nil })
}

// ActionOpts carries the disable reason and per-request headers.
type ActionOpts struct {
	Reason  string
	Headers map[string]string
}

func copyAction(v ActionOpts) ActionOpts { v.Headers = maps.Clone(v.Headers); return v }

// EnableOption is one option of Enable: request.Headers (replaces the
// headers) or request.Custom over *ActionOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type EnableOption = request.WriteCall

// DisableOption is one option of Disable: WithDisableReason, request.Headers
// (replaces the headers) or request.Custom over *ActionOpts. Other shared
// calls fail with resource.ErrInvalidOption before HTTP. An empty reason is
// not sent, as in Python.
type DisableOption = request.WriteCall

func WithDisableReason(reason string) DisableOption {
	return request.Custom(func(v *ActionOpts) error { v.Reason = reason; return nil })
}

var (
	getCalls    = calls.Spec[GetOpts]{Headers: cinderowned.ReadHeadersCall}
	findCalls   = calls.Spec[FindOpts]{IgnoreMissing: cinderowned.FindIgnoreMissingCall}
	actionCalls = calls.Spec[ActionOpts]{Headers: calls.Value(func(v *ActionOpts) *map[string]string { return &v.Headers })}
	listCalls   = calls.Spec[ListOpts]{
		Headers: cinderowned.ListHeadersCall(inner),
		Query:   cinderowned.ListQueryCall(spec, inner),
		Paging:  cinderowned.ListPaging(inner),
	}
)
