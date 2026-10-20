package federationprotocols

import "github.com/JSYoo5B/go-openstacksdk/identity/v3/internal/osfederation"

// GetOpts carries per-request headers, a Go extension.
type GetOpts = osfederation.ReadOpts
type GetOption func(*GetOpts) error

func WithGetHeaders(values map[string]string) GetOption {
	return GetOption(osfederation.ReadHeaders(values))
}

// CreateOpts carries the Python **attrs as raw JSON. Keys are Python
// attribute names (name, mapping_id) or Body keys (id, mapping_id);
// unknown keys are dropped and id/name never enter the body.
type CreateOpts = osfederation.WriteOpts
type CreateOption func(*CreateOpts) error

func WithCreateOpts(opts CreateOpts) CreateOption {
	return CreateOption(osfederation.WriteSnapshot(opts))
}
func WithCreateField(key string, value any) CreateOption {
	return CreateOption(spec.WriteField(key, value))
}
func WithCreateHeaders(values map[string]string) CreateOption {
	return CreateOption(osfederation.WriteHeaders(values))
}

// UpdateOpts carries the Python **attrs of update_federation_protocol.
type UpdateOpts = osfederation.WriteOpts
type UpdateOption func(*UpdateOpts) error

func WithUpdateOpts(opts UpdateOpts) UpdateOption {
	return UpdateOption(osfederation.WriteSnapshot(opts))
}
func WithUpdateField(key string, value any) UpdateOption {
	return UpdateOption(spec.WriteField(key, value))
}
func WithUpdateHeaders(values map[string]string) UpdateOption {
	return UpdateOption(osfederation.WriteHeaders(values))
}

// DeleteOpts selects ignore_missing; nil keeps the Python default true.
type DeleteOpts = osfederation.DeleteOpts
type DeleteOption func(*DeleteOpts) error

func WithDeleteIgnoreMissing(value bool) DeleteOption {
	return DeleteOption(osfederation.DeleteIgnoreMissing(value))
}
func WithDeleteHeaders(values map[string]string) DeleteOption {
	return DeleteOption(osfederation.DeleteHeaders(values))
}

// FindOpts selects ignore_missing; nil keeps the Python default true.
type FindOpts = osfederation.FindOpts
type FindOption func(*FindOpts) error

func WithFindIgnoreMissing(value bool) FindOption {
	return FindOption(osfederation.FindIgnoreMissing(value))
}

// ListOpts carries Python **query: id goes to the server; name and
// mapping_id filter locally; other names are ignored.
type ListOpts = osfederation.ListOpts
type ListOption func(*ListOpts) error

func WithListOpts(opts ListOpts) ListOption { return ListOption(osfederation.ListSnapshot(opts)) }
func WithListQuery(key string, value any) ListOption {
	return ListOption(spec.ListQuery(key, value))
}
func WithListID(id string) ListOption         { return WithListQuery("id", id) }
func WithListLimit(limit int) ListOption      { return ListOption(osfederation.ListLimit(limit)) }
func WithListMarker(marker string) ListOption { return ListOption(osfederation.ListMarker(marker)) }
func WithListMaxItems(count int) ListOption   { return ListOption(osfederation.ListMaxItems(count)) }
func WithListPaginated(value bool) ListOption { return ListOption(osfederation.ListPaginated(value)) }
func WithListHeaders(values map[string]string) ListOption {
	return ListOption(osfederation.ListHeaders(values))
}
