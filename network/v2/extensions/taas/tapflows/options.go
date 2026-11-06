package tapflows

import (
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/internal/ownedrecord"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// CreateOpts carries the Python **attrs of create_tap_flow as raw JSON.
// Keys are Python attribute names or wire names; unknown keys stay in the
// body. Headers and Microversion are Go extensions.
type CreateOpts = ownedrecord.WriteOpts

// CreateOption is one option of Create: WithCreateOpts, request.Field (one
// Python attribute as JSON), request.Headers (replaces the headers),
// request.Microversion or request.Custom over *CreateOpts. Other shared
// calls fail with resource.ErrInvalidOption before HTTP.
type CreateOption = request.WriteCall

func WithCreateOpts(opts CreateOpts) CreateOption {
	return request.Custom(ownedrecord.WriteSnapshot(opts))
}

// GetOpts carries the get_tap_flow fields selector.
type GetOpts = ownedrecord.GetOpts

// GetOption is one option of Get: WithGetOpts, WithGetFields,
// request.Headers (replaces the headers), request.Microversion or
// request.Custom over *GetOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type GetOption = request.GetCall

func WithGetOpts(opts GetOpts) GetOption { return request.Custom(ownedrecord.GetSnapshot(opts)) }
func WithGetFields(fields ...string) GetOption {
	return request.Custom(ownedrecord.GetFields(fields...))
}

// UpdateOpts carries the Python **attrs of update_tap_flow. The id and
// tap_flow names are bound arguments and are rejected.
type UpdateOpts = ownedrecord.WriteOpts

// UpdateOption is one option of Update: WithUpdateOpts, request.Field (one
// Python attribute as JSON), request.Headers (replaces the headers),
// request.Microversion or request.Custom over *UpdateOpts. Other shared
// calls fail with resource.ErrInvalidOption before HTTP.
type UpdateOption = request.WriteCall

func WithUpdateOpts(opts UpdateOpts) UpdateOption {
	return request.Custom(ownedrecord.WriteSnapshot(opts))
}

// DeleteOpts selects ignore_missing; nil keeps the Python default true.
type DeleteOpts = ownedrecord.DeleteOpts

// DeleteOption is one option of Delete: WithDeleteOpts,
// request.IgnoreMissing, request.Headers (replaces the headers),
// request.Microversion or request.Custom over *DeleteOpts. Other shared
// calls fail with resource.ErrInvalidOption before HTTP.
type DeleteOption = request.DeleteCall

func WithDeleteOpts(opts DeleteOpts) DeleteOption {
	return request.Custom(ownedrecord.DeleteSnapshot(opts))
}

// FindOpts selects ignore_missing (nil keeps true) and carries **query.
type FindOpts = ownedrecord.FindOpts

// FindOption is one option of Find: WithFindOpts, request.IgnoreMissing,
// request.Query (one Python **query value as JSON), request.Headers
// (replaces the headers), request.Microversion or request.Custom over
// *FindOpts. Other shared calls fail with resource.ErrInvalidOption before
// HTTP.
type FindOption = request.FindCall

func WithFindOpts(opts FindOpts) FindOption { return request.Custom(ownedrecord.FindSnapshot(opts)) }

// ListOpts carries Python **query: sort_key, sort_dir, fields, name,
// project_id, limit and marker go to the server; id, description, tenant_id,
// tap_service_id, direction, status and source_port filter locally; other
// names are ignored.
type ListOpts = ownedrecord.ListOpts

// ListOption is one option of List and All: WithListOpts, request.Query (one
// Python **query value as JSON; names the list discards are dropped),
// request.Limit, request.Marker, request.MaxItems, request.Paginated,
// request.Headers (replaces the headers), request.Microversion or
// request.Custom over *ListOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type ListOption = request.ListCall

func WithListOpts(opts ListOpts) ListOption { return request.Custom(ownedrecord.ListSnapshot(opts)) }
