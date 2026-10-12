package portchains

import "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/internal/sfcrecord"

// GetOpts carries per-request headers, a Go extension.
type GetOpts = sfcrecord.ReadOpts
type GetOption func(*GetOpts) error

func WithGetHeaders(values map[string]string) GetOption {
	return GetOption(sfcrecord.ReadHeaders(values))
}

// CreateOpts carries the Python **attrs as raw JSON. Keys are Python
// attribute names or Body keys; unknown keys are dropped.
type CreateOpts = sfcrecord.WriteOpts
type CreateOption func(*CreateOpts) error

func WithCreateOpts(opts CreateOpts) CreateOption {
	return CreateOption(sfcrecord.WriteSnapshot(opts))
}
func WithCreateField(key string, value any) CreateOption {
	return CreateOption(spec.WriteField(key, value))
}
func WithCreateHeaders(values map[string]string) CreateOption {
	return CreateOption(sfcrecord.WriteHeaders(values))
}

// UpdateOpts carries the Python **attrs of update_sfc_port_chain; id is never sent.
type UpdateOpts = sfcrecord.WriteOpts
type UpdateOption func(*UpdateOpts) error

func WithUpdateOpts(opts UpdateOpts) UpdateOption {
	return UpdateOption(sfcrecord.WriteSnapshot(opts))
}
func WithUpdateField(key string, value any) UpdateOption {
	return UpdateOption(spec.WriteField(key, value))
}
func WithUpdateHeaders(values map[string]string) UpdateOption {
	return UpdateOption(sfcrecord.WriteHeaders(values))
}

// DeleteOpts selects ignore_missing; nil keeps the Python default true.
type DeleteOpts = sfcrecord.DeleteOpts
type DeleteOption func(*DeleteOpts) error

func WithDeleteIgnoreMissing(value bool) DeleteOption {
	return DeleteOption(sfcrecord.DeleteIgnoreMissing(value))
}
func WithDeleteHeaders(values map[string]string) DeleteOption {
	return DeleteOption(sfcrecord.DeleteHeaders(values))
}

// FindOpts selects ignore_missing (nil keeps the Python default true) and
// carries the Python find **query.
type FindOpts = sfcrecord.FindOpts
type FindOption func(*FindOpts) error

func WithFindIgnoreMissing(value bool) FindOption {
	return FindOption(sfcrecord.FindIgnoreMissing(value))
}

// WithFindQuery adds one Python find **query name. It is sent on the GET by
// ID and passed to the list fallback; id and Resource controls are rejected.
func WithFindQuery(key string, value any) FindOption {
	return FindOption(sfcrecord.FindQuery(key, value))
}

// ListOpts carries Python **query: description, name, project_id, tenant_id,
// limit and marker go to the server; the other Body attributes
// (chain_parameters, flow_classifiers, id, port_pair_groups) filter locally; other names are ignored.
type ListOpts = sfcrecord.ListOpts
type ListOption func(*ListOpts) error

func WithListOpts(opts ListOpts) ListOption { return ListOption(sfcrecord.ListSnapshot(opts)) }
func WithListQuery(key string, value any) ListOption {
	return ListOption(spec.ListQuery(key, value))
}
func WithListName(name string) ListOption           { return WithListQuery("name", name) }
func WithListProjectID(projectID string) ListOption { return WithListQuery("project_id", projectID) }
func WithListLimit(limit int) ListOption            { return ListOption(sfcrecord.ListLimit(limit)) }
func WithListMarker(marker string) ListOption       { return ListOption(sfcrecord.ListMarker(marker)) }
func WithListMaxItems(count int) ListOption         { return ListOption(sfcrecord.ListMaxItems(count)) }
func WithListPaginated(value bool) ListOption       { return ListOption(sfcrecord.ListPaginated(value)) }
func WithListHeaders(values map[string]string) ListOption {
	return ListOption(sfcrecord.ListHeaders(values))
}
