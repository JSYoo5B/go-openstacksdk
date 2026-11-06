package rules

import (
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/internal/ownedrecords"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// GetOpts carries the Python fields selector; names map to wire keys
// (is_excluded becomes excluded). Headers and Microversion are Go extensions.
type GetOpts = ownedrecords.GetOpts

// GetOption is one option of Get: WithGetOpts, WithGetFields,
// request.Headers (replaces the headers), request.Microversion or
// request.Custom over *GetOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type GetOption = request.GetCall

func WithGetOpts(opts GetOpts) GetOption { return request.Custom(ownedrecords.GetSnapshot(opts)) }
func WithGetFields(fields ...string) GetOption {
	return request.Custom(ownedrecords.GetFields(fields...))
}

// CreateOpts carries Python **attrs as raw JSON. Keys are Python names
// (is_excluded, metering_label_id) or wire keys; unknown keys are sent.
type CreateOpts = ownedrecords.WriteOpts

// CreateOption is one option of Create: WithCreateOpts, request.Field (one
// Python attribute as JSON), request.Headers (replaces the headers),
// request.Microversion or request.Custom over *CreateOpts. Other shared
// calls fail with resource.ErrInvalidOption before HTTP.
type CreateOption = request.WriteCall

func WithCreateOpts(opts CreateOpts) CreateOption {
	return request.Custom(ownedrecords.WriteSnapshot(opts))
}

// UpdateOpts carries the Python **attrs of update_metering_label_rule.
// id and metering_label_rule cannot be attributes because the ID is the method
// argument.
type UpdateOpts = ownedrecords.WriteOpts

// UpdateOption is one option of Update: WithUpdateOpts, request.Field (one
// Python attribute as JSON), request.Headers (replaces the headers),
// request.Microversion or request.Custom over *UpdateOpts. Other shared
// calls fail with resource.ErrInvalidOption before HTTP.
type UpdateOption = request.WriteCall

func WithUpdateOpts(opts UpdateOpts) UpdateOption {
	return request.Custom(ownedrecords.WriteSnapshot(opts))
}

// DeleteOpts selects ignore_missing; nil keeps the Python default true.
type DeleteOpts = ownedrecords.DeleteOpts

// DeleteOption is one option of Delete: WithDeleteOpts,
// request.IgnoreMissing, request.Headers (replaces the headers),
// request.Microversion or request.Custom over *DeleteOpts. Other shared
// calls fail with resource.ErrInvalidOption before HTTP.
type DeleteOption = request.DeleteCall

func WithDeleteOpts(opts DeleteOpts) DeleteOption {
	return request.Custom(ownedrecords.DeleteSnapshot(opts))
}

// ListOpts carries Python **query: limit, marker, direction,
// fields, metering_label_id, remote_ip_prefix, source_ip_prefix,
// destination_ip_prefix, project_id, sort_key and sort_dir go to the server;
// id, name, is_excluded and tenant_id filter rows locally; max_items and
// paginated control paging; other names are ignored.
type ListOpts = ownedrecords.ListOpts

// ListOption is one option of List and All: WithListOpts, request.Query (one
// Python **query value as JSON; names the list discards are dropped),
// request.Limit, request.Marker, request.MaxItems, request.Paginated,
// request.Headers (replaces the headers), request.Microversion or
// request.Custom over *ListOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type ListOption = request.ListCall

func WithListOpts(opts ListOpts) ListOption { return request.Custom(ownedrecords.ListSnapshot(opts)) }

// FindOpts carries ignore_missing (nil means true) and Python **query, which
// goes to the member GET and then to the fallback list.
type FindOpts = ownedrecords.FindOpts

// FindOption is one option of Find: WithFindOpts, request.IgnoreMissing,
// request.Query (one Python **query value as JSON), request.Headers
// (replaces the headers), request.Microversion or request.Custom over
// *FindOpts. Other shared calls fail with resource.ErrInvalidOption before
// HTTP.
type FindOption = request.FindCall

func WithFindOpts(opts FindOpts) FindOption { return request.Custom(ownedrecords.FindSnapshot(opts)) }
