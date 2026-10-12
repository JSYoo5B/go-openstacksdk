package clusters

import (
	"maps"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/internal/cinderowned"
)

// GetOpts carries per-request headers, a Go extension.
type GetOpts = cinderowned.ReadOpts
type GetOption func(*GetOpts) error

func WithGetHeaders(values map[string]string) GetOption {
	return GetOption(cinderowned.ReadHeaders(values))
}

// FindOpts selects ignore_missing; nil keeps the Python default true.
type FindOpts = cinderowned.FindOpts
type FindOption func(*FindOpts) error

func WithFindIgnoreMissing(value bool) FindOption {
	return FindOption(cinderowned.FindIgnoreMissing(value))
}

// ListOpts carries Python details (nil means true) and **query: name,
// binary, is_up, disabled, num_hosts, num_down_hosts, replication_status,
// frozen and active_backend_id go to the server; id, created_at,
// disabled_reason, last_heartbeat, state, status and updated_at filter
// locally; other names are ignored.
type ListOpts struct {
	cinderowned.ListOpts
	Details *bool
}
type ListOption func(*ListOpts) error

func copyList(v ListOpts) ListOpts {
	v.ListOpts = cinderowned.CopyList(v.ListOpts)
	if v.Details != nil {
		x := *v.Details
		v.Details = &x
	}
	return v
}

func inner(apply func(*cinderowned.ListOpts) error) ListOption {
	return func(v *ListOpts) error { return apply(&v.ListOpts) }
}

func WithListOpts(opts ListOpts) ListOption {
	owned := copyList(opts)
	return func(v *ListOpts) error { *v = copyList(owned); return nil }
}
func WithListDetails(value bool) ListOption {
	return func(v *ListOpts) error { x := value; v.Details = &x; return nil }
}
func WithListQuery(key string, value any) ListOption { return inner(spec.ListQuery(key, value)) }
func WithListLimit(limit int) ListOption             { return inner(cinderowned.ListLimit(limit)) }
func WithListMarker(marker string) ListOption        { return inner(cinderowned.ListMarker(marker)) }
func WithListMaxItems(count int) ListOption          { return inner(cinderowned.ListMaxItems(count)) }
func WithListPaginated(value bool) ListOption        { return inner(cinderowned.ListPaginated(value)) }
func WithListHeaders(values map[string]string) ListOption {
	return inner(cinderowned.ListHeaders(values))
}

// ActionOpts carries the disable reason and per-request headers.
type ActionOpts struct {
	Reason  string
	Headers map[string]string
}

func copyAction(v ActionOpts) ActionOpts { v.Headers = maps.Clone(v.Headers); return v }

// EnableOption configures Enable.
type EnableOption func(*ActionOpts) error

func WithEnableHeaders(values map[string]string) EnableOption {
	owned := maps.Clone(values)
	return func(v *ActionOpts) error { v.Headers = maps.Clone(owned); return nil }
}

// DisableOption configures Disable. An empty reason is not sent, as in Python.
type DisableOption func(*ActionOpts) error

func WithDisableReason(reason string) DisableOption {
	return func(v *ActionOpts) error { v.Reason = reason; return nil }
}
func WithDisableHeaders(values map[string]string) DisableOption {
	owned := maps.Clone(values)
	return func(v *ActionOpts) error { v.Headers = maps.Clone(owned); return nil }
}
