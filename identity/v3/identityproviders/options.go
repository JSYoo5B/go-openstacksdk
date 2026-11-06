package identityproviders

import (
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/internal/osfederation"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// GetOpts carries per-request headers, a Go extension.
type GetOpts = osfederation.ReadOpts

// GetOption is one option of Get: request.Headers (replaces the headers) or
// request.Custom over *GetOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type GetOption = request.GetCall

// CreateOpts carries the Python **attrs as raw JSON. Keys are Python
// attribute names (name, is_enabled, ...) or Body keys (id, enabled, ...);
// unknown keys are dropped and id/name never enter the body.
type CreateOpts = osfederation.WriteOpts

// CreateOption is one option of Create: WithCreateOpts, request.Field (one
// Python attribute; unknown names are dropped with their marshal error),
// request.Headers (replaces the headers) or request.Custom over *CreateOpts.
// Other shared calls fail with resource.ErrInvalidOption before HTTP.
type CreateOption = request.WriteCall

func WithCreateOpts(opts CreateOpts) CreateOption {
	return request.Custom(osfederation.WriteSnapshot(opts))
}

// UpdateOpts carries the Python **attrs of update_identity_provider.
type UpdateOpts = osfederation.WriteOpts

// UpdateOption is one option of Update: WithUpdateOpts, request.Field (one
// Python attribute; unknown names are dropped with their marshal error),
// request.Headers (replaces the headers) or request.Custom over *UpdateOpts.
// Other shared calls fail with resource.ErrInvalidOption before HTTP.
type UpdateOption = request.WriteCall

func WithUpdateOpts(opts UpdateOpts) UpdateOption {
	return request.Custom(osfederation.WriteSnapshot(opts))
}

// DeleteOpts selects ignore_missing; nil keeps the Python default true.
type DeleteOpts = osfederation.DeleteOpts

// DeleteOption is one option of Delete: request.IgnoreMissing,
// request.Headers (replaces the headers) or request.Custom over *DeleteOpts.
// Other shared calls fail with resource.ErrInvalidOption before HTTP.
type DeleteOption = request.DeleteCall

// FindOpts selects ignore_missing; nil keeps the Python default true.
type FindOpts = osfederation.FindOpts

// FindOption is one option of Find: request.IgnoreMissing or request.Custom
// over *FindOpts. Other shared calls fail with resource.ErrInvalidOption
// before HTTP.
type FindOption = request.FindCall

// ListOpts carries Python **query: id, is_enabled (or enabled) go to the
// server; name, description, domain_id, remote_ids and authorization_ttl
// filter locally; other names are ignored.
type ListOpts = osfederation.ListOpts

// ListOption is one option of List and All: WithListOpts, WithListID, WithListEnabled,
// request.Query (one Python **query name as JSON; unknown names are dropped
// with their marshal error), request.Limit, request.Marker, request.MaxItems,
// request.Paginated, request.Headers (replaces the headers) or request.Custom
// over *ListOpts. Other shared calls fail with resource.ErrInvalidOption
// before HTTP.
type ListOption = request.ListCall

func WithListOpts(opts ListOpts) ListOption { return request.Custom(osfederation.ListSnapshot(opts)) }
func WithListID(id string) ListOption       { return request.Custom(spec.ListQuery("id", id)) }
func WithListEnabled(enabled bool) ListOption {
	return request.Custom(spec.ListQuery("is_enabled", enabled))
}

var (
	writeCalls = spec.WriteCalls()
	listCalls  = spec.ListCalls()
)
