package grouptypes

import "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/internal/cinderowned"

// GetOpts carries per-request headers, a Go extension. It is shared by Get
// and the group_specs calls other than DeleteGroupSpecsProperty.
type GetOpts = cinderowned.ReadOpts
type GetOption func(*GetOpts) error

func WithGetHeaders(values map[string]string) GetOption {
	return GetOption(cinderowned.ReadHeaders(values))
}

// CreateOpts carries the Python **attrs as raw JSON. Keys are the declared
// attributes id, name, description, group_specs and is_public; unknown keys
// are dropped.
type CreateOpts = cinderowned.WriteOpts
type CreateOption func(*CreateOpts) error

func WithCreateOpts(opts CreateOpts) CreateOption {
	return CreateOption(cinderowned.WriteSnapshot(opts))
}
func WithCreateField(key string, value any) CreateOption {
	return CreateOption(spec.WriteField(key, value))
}
func WithCreateHeaders(values map[string]string) CreateOption {
	return CreateOption(cinderowned.WriteHeaders(values))
}

// UpdateOpts carries the Python **attrs of update_group_type; id is never
// sent.
type UpdateOpts = cinderowned.WriteOpts
type UpdateOption func(*UpdateOpts) error

func WithUpdateOpts(opts UpdateOpts) UpdateOption {
	return UpdateOption(cinderowned.WriteSnapshot(opts))
}
func WithUpdateField(key string, value any) UpdateOption {
	return UpdateOption(spec.WriteField(key, value))
}
func WithUpdateHeaders(values map[string]string) UpdateOption {
	return UpdateOption(cinderowned.WriteHeaders(values))
}

// DeleteOpts selects ignore_missing; nil keeps the Python default true.
type DeleteOpts = cinderowned.DeleteOpts
type DeleteOption func(*DeleteOpts) error

func WithDeleteIgnoreMissing(value bool) DeleteOption {
	return DeleteOption(cinderowned.DeleteIgnoreMissing(value))
}
func WithDeleteHeaders(values map[string]string) DeleteOption {
	return DeleteOption(cinderowned.DeleteHeaders(values))
}

// FindOpts selects ignore_missing; nil keeps the Python default true.
type FindOpts = cinderowned.FindOpts
type FindOption func(*FindOpts) error

func WithFindIgnoreMissing(value bool) FindOption {
	return FindOption(cinderowned.FindIgnoreMissing(value))
}

// ListOpts carries Python **query: is_public, offset, sort_dir, sort_key,
// sort and all_projects (sent as all_tenants) go to the server; id, name,
// description and group_specs filter locally; other names are ignored.
type ListOpts = cinderowned.ListOpts
type ListOption func(*ListOpts) error

func WithListOpts(opts ListOpts) ListOption { return ListOption(cinderowned.ListSnapshot(opts)) }
func WithListQuery(key string, value any) ListOption {
	return ListOption(spec.ListQuery(key, value))
}
func WithListLimit(limit int) ListOption      { return ListOption(cinderowned.ListLimit(limit)) }
func WithListMarker(marker string) ListOption { return ListOption(cinderowned.ListMarker(marker)) }
func WithListMaxItems(count int) ListOption   { return ListOption(cinderowned.ListMaxItems(count)) }
func WithListPaginated(value bool) ListOption { return ListOption(cinderowned.ListPaginated(value)) }
func WithListHeaders(values map[string]string) ListOption {
	return ListOption(cinderowned.ListHeaders(values))
}
