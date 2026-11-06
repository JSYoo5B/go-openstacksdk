package portchains

import (
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/internal/sfcrecord"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// GetOpts carries per-request headers, a Go extension.
type GetOpts = sfcrecord.ReadOpts

// GetOption is one option of Get: request.Headers (replaces the headers) or
// request.Custom over *GetOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type GetOption = request.GetCall

// CreateOpts carries the Python **attrs as raw JSON. Keys are Python
// attribute names or Body keys; unknown keys are dropped.
type CreateOpts = sfcrecord.WriteOpts

// CreateOption is one option of Create: WithCreateOpts, request.Field (one
// Python attribute; unknown names are dropped with their marshal error),
// request.Headers (replaces the headers) or request.Custom over *CreateOpts.
// Other shared calls fail with resource.ErrInvalidOption before HTTP.
type CreateOption = request.WriteCall

func WithCreateOpts(opts CreateOpts) CreateOption {
	return request.Custom(sfcrecord.WriteSnapshot(opts))
}

// UpdateOpts carries the Python **attrs of update_sfc_port_chain; id is never sent.
type UpdateOpts = sfcrecord.WriteOpts

// UpdateOption is one option of Update: WithUpdateOpts, request.Field (one
// Python attribute; unknown names are dropped with their marshal error),
// request.Headers (replaces the headers) or request.Custom over *UpdateOpts.
// Other shared calls fail with resource.ErrInvalidOption before HTTP.
type UpdateOption = request.WriteCall

func WithUpdateOpts(opts UpdateOpts) UpdateOption {
	return request.Custom(sfcrecord.WriteSnapshot(opts))
}

// DeleteOpts selects ignore_missing; nil keeps the Python default true.
type DeleteOpts = sfcrecord.DeleteOpts

// DeleteOption is one option of Delete: request.IgnoreMissing,
// request.Headers (replaces the headers) or request.Custom over *DeleteOpts.
// Other shared calls fail with resource.ErrInvalidOption before HTTP.
type DeleteOption = request.DeleteCall

// FindOpts selects ignore_missing (nil keeps the Python default true) and
// carries the Python find **query.
type FindOpts = sfcrecord.FindOpts

// FindOption is one option of Find: request.IgnoreMissing, request.Query
// (one Python find **query name, sent on the GET by ID and passed to the
// list fallback; id and Resource controls are rejected when Find runs) or
// request.Custom over *FindOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type FindOption = request.FindCall

// ListOpts carries Python **query: description, name, project_id, tenant_id,
// limit and marker go to the server; the other Body attributes
// (chain_parameters, flow_classifiers, id, port_pair_groups) filter locally; other names are ignored.
type ListOpts = sfcrecord.ListOpts

// ListOption is one option of List and All: WithListOpts, WithListName,
// WithListProjectID, request.Query (one Python **query name as JSON; unknown
// names are dropped with their marshal error), request.Limit,
// request.Marker, request.MaxItems, request.Paginated, request.Headers
// (replaces the headers) or request.Custom over *ListOpts. Other shared
// calls fail with resource.ErrInvalidOption before HTTP.
type ListOption = request.ListCall

func WithListOpts(opts ListOpts) ListOption { return request.Custom(sfcrecord.ListSnapshot(opts)) }
func WithListName(name string) ListOption   { return request.Custom(spec.ListQuery("name", name)) }
func WithListProjectID(projectID string) ListOption {
	return request.Custom(spec.ListQuery("project_id", projectID))
}

var (
	writeCalls = spec.WriteCalls()
	listCalls  = spec.ListCalls()
)
