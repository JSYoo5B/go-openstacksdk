package labels

import "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/internal/ownedrecords"

// GetOpts carries the Python fields selector; names map to wire keys
// (is_shared becomes shared). Headers and Microversion are Go extensions.
type GetOpts = ownedrecords.GetOpts
type GetOption func(*GetOpts) error

func WithGetOpts(opts GetOpts) GetOption       { return GetOption(ownedrecords.GetSnapshot(opts)) }
func WithGetFields(fields ...string) GetOption { return GetOption(ownedrecords.GetFields(fields...)) }
func WithGetHeaders(values map[string]string) GetOption {
	return GetOption(ownedrecords.GetHeaders(values))
}
func WithGetMicroversion(version string) GetOption {
	return GetOption(ownedrecords.GetMicroversion(version))
}

// CreateOpts carries Python **attrs as raw JSON. Keys are Python names
// (is_shared, project_id) or wire keys; unknown keys are sent.
type CreateOpts = ownedrecords.WriteOpts
type CreateOption func(*CreateOpts) error

func WithCreateOpts(opts CreateOpts) CreateOption {
	return CreateOption(ownedrecords.WriteSnapshot(opts))
}
func WithCreateField(key string, value any) CreateOption {
	return CreateOption(ownedrecords.WriteField(key, value))
}
func WithCreateHeaders(values map[string]string) CreateOption {
	return CreateOption(ownedrecords.WriteHeaders(values))
}
func WithCreateMicroversion(version string) CreateOption {
	return CreateOption(ownedrecords.WriteMicroversion(version))
}

// UpdateOpts carries the Python **attrs of update_metering_label.
// id and metering_label cannot be attributes because the ID is the method
// argument.
type UpdateOpts = ownedrecords.WriteOpts
type UpdateOption func(*UpdateOpts) error

func WithUpdateOpts(opts UpdateOpts) UpdateOption {
	return UpdateOption(ownedrecords.WriteSnapshot(opts))
}
func WithUpdateField(key string, value any) UpdateOption {
	return UpdateOption(ownedrecords.WriteField(key, value))
}
func WithUpdateHeaders(values map[string]string) UpdateOption {
	return UpdateOption(ownedrecords.WriteHeaders(values))
}
func WithUpdateMicroversion(version string) UpdateOption {
	return UpdateOption(ownedrecords.WriteMicroversion(version))
}

// DeleteOpts selects ignore_missing; nil keeps the Python default true.
type DeleteOpts = ownedrecords.DeleteOpts
type DeleteOption func(*DeleteOpts) error

func WithDeleteOpts(opts DeleteOpts) DeleteOption {
	return DeleteOption(ownedrecords.DeleteSnapshot(opts))
}
func WithDeleteIgnoreMissing(value bool) DeleteOption {
	return DeleteOption(ownedrecords.DeleteIgnoreMissing(value))
}
func WithDeleteHeaders(values map[string]string) DeleteOption {
	return DeleteOption(ownedrecords.DeleteHeaders(values))
}
func WithDeleteMicroversion(version string) DeleteOption {
	return DeleteOption(ownedrecords.DeleteMicroversion(version))
}

// ListOpts carries Python **query: limit, marker, description,
// fields, name, project_id, sort_key, sort_dir and is_shared (shared) go to
// the server; id and tenant_id filter rows locally; max_items and paginated
// control paging; other names are ignored.
type ListOpts = ownedrecords.ListOpts
type ListOption func(*ListOpts) error

func WithListOpts(opts ListOpts) ListOption { return ListOption(ownedrecords.ListSnapshot(opts)) }
func WithListQuery(key string, value any) ListOption {
	return ListOption(spec.ListQuery(key, value))
}
func WithListLimit(limit int) ListOption      { return WithListQuery("limit", limit) }
func WithListMarker(marker string) ListOption { return WithListQuery("marker", marker) }
func WithListMaxItems(count int) ListOption   { return WithListQuery("max_items", count) }
func WithListPaginated(value bool) ListOption { return WithListQuery("paginated", value) }
func WithListHeaders(values map[string]string) ListOption {
	return ListOption(ownedrecords.ListHeaders(values))
}
func WithListMicroversion(version string) ListOption {
	return ListOption(ownedrecords.ListMicroversion(version))
}

// FindOpts carries ignore_missing (nil means true) and Python **query, which
// goes to the member GET and then to the fallback list.
type FindOpts = ownedrecords.FindOpts
type FindOption func(*FindOpts) error

func WithFindOpts(opts FindOpts) FindOption { return FindOption(ownedrecords.FindSnapshot(opts)) }
func WithFindIgnoreMissing(value bool) FindOption {
	return FindOption(ownedrecords.FindIgnoreMissing(value))
}
func WithFindQuery(key string, value any) FindOption {
	return FindOption(ownedrecords.FindQuery(key, value))
}
func WithFindHeaders(values map[string]string) FindOption {
	return FindOption(ownedrecords.FindHeaders(values))
}
func WithFindMicroversion(version string) FindOption {
	return FindOption(ownedrecords.FindMicroversion(version))
}
