package localips

import "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/internal/ownedrecord"

// CreateOpts carries the Python **attrs of create_local_ip as raw JSON.
// Keys are Python attribute names or wire names; unknown keys stay in the
// body. Headers and Microversion are Go extensions.
type CreateOpts = ownedrecord.WriteOpts
type CreateOption func(*CreateOpts) error

func WithCreateOpts(opts CreateOpts) CreateOption {
	return CreateOption(ownedrecord.WriteSnapshot(opts))
}
func WithCreateField(key string, value any) CreateOption {
	return CreateOption(ownedrecord.WriteField(key, value))
}
func WithCreateHeaders(values map[string]string) CreateOption {
	return CreateOption(ownedrecord.WriteHeaders(values))
}
func WithCreateMicroversion(version string) CreateOption {
	return CreateOption(ownedrecord.WriteMicroversion(version))
}

// GetOpts carries the get_local_ip fields selector.
type GetOpts = ownedrecord.GetOpts
type GetOption func(*GetOpts) error

func WithGetOpts(opts GetOpts) GetOption { return GetOption(ownedrecord.GetSnapshot(opts)) }
func WithGetFields(fields ...string) GetOption {
	return GetOption(ownedrecord.GetFields(fields...))
}
func WithGetHeaders(values map[string]string) GetOption {
	return GetOption(ownedrecord.GetHeaders(values))
}
func WithGetMicroversion(version string) GetOption {
	return GetOption(ownedrecord.GetMicroversion(version))
}

// UpdateOpts carries the Python **attrs of update_local_ip. The id,
// local_ip and if_revision names are bound arguments and are rejected.
type UpdateOpts = ownedrecord.WriteOpts
type UpdateOption func(*UpdateOpts) error

func WithUpdateOpts(opts UpdateOpts) UpdateOption {
	return UpdateOption(ownedrecord.WriteSnapshot(opts))
}
func WithUpdateField(key string, value any) UpdateOption {
	return UpdateOption(ownedrecord.WriteField(key, value))
}
func WithUpdateHeaders(values map[string]string) UpdateOption {
	return UpdateOption(ownedrecord.WriteHeaders(values))
}
func WithUpdateMicroversion(version string) UpdateOption {
	return UpdateOption(ownedrecord.WriteMicroversion(version))
}

// DeleteOpts selects ignore_missing; nil keeps the Python default true.
type DeleteOpts = ownedrecord.DeleteOpts
type DeleteOption func(*DeleteOpts) error

func WithDeleteOpts(opts DeleteOpts) DeleteOption {
	return DeleteOption(ownedrecord.DeleteSnapshot(opts))
}
func WithDeleteIgnoreMissing(value bool) DeleteOption {
	return DeleteOption(ownedrecord.DeleteIgnoreMissing(value))
}
func WithDeleteHeaders(values map[string]string) DeleteOption {
	return DeleteOption(ownedrecord.DeleteHeaders(values))
}
func WithDeleteMicroversion(version string) DeleteOption {
	return DeleteOption(ownedrecord.DeleteMicroversion(version))
}

// FindOpts selects ignore_missing (nil keeps true) and carries **query.
type FindOpts = ownedrecord.FindOpts
type FindOption func(*FindOpts) error

func WithFindOpts(opts FindOpts) FindOption { return FindOption(ownedrecord.FindSnapshot(opts)) }
func WithFindIgnoreMissing(value bool) FindOption {
	return FindOption(ownedrecord.FindIgnoreMissing(value))
}
func WithFindQuery(key string, value any) FindOption {
	return FindOption(ownedrecord.FindQuery(key, value))
}
func WithFindHeaders(values map[string]string) FindOption {
	return FindOption(ownedrecord.FindHeaders(values))
}
func WithFindMicroversion(version string) FindOption {
	return FindOption(ownedrecord.FindMicroversion(version))
}

// ListOpts carries Python **query: sort_key, sort_dir, name, description,
// fields, project_id, network_id, local_port_id, local_ip_address, ip_mode,
// limit and marker go to the server; id, created_at, revision_number and
// updated_at filter locally; other names are ignored.
type ListOpts = ownedrecord.ListOpts
type ListOption func(*ListOpts) error

func WithListOpts(opts ListOpts) ListOption { return ListOption(ownedrecord.ListSnapshot(opts)) }
func WithListQuery(key string, value any) ListOption {
	return ListOption(spec.ListQuery(key, value))
}
func WithListLimit(limit int) ListOption      { return WithListQuery("limit", limit) }
func WithListMarker(marker string) ListOption { return WithListQuery("marker", marker) }
func WithListMaxItems(count int) ListOption   { return WithListQuery("max_items", count) }
func WithListPaginated(value bool) ListOption { return WithListQuery("paginated", value) }
func WithListHeaders(values map[string]string) ListOption {
	return ListOption(ownedrecord.ListHeaders(values))
}
func WithListMicroversion(version string) ListOption {
	return ListOption(ownedrecord.ListMicroversion(version))
}
