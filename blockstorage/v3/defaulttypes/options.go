package defaulttypes

import "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/internal/cinderowned"

// Opts carries per-request headers for Get and Set, a Go extension.
type Opts = cinderowned.ReadOpts
type Option func(*Opts) error

func WithHeaders(values map[string]string) Option {
	return Option(cinderowned.ReadHeaders(values))
}

// DeleteOpts selects ignore_missing for Unset; nil keeps the Python
// behavior of ignoring a 404.
type DeleteOpts = cinderowned.DeleteOpts
type DeleteOption func(*DeleteOpts) error

func WithDeleteIgnoreMissing(value bool) DeleteOption {
	return DeleteOption(cinderowned.DeleteIgnoreMissing(value))
}
func WithDeleteHeaders(values map[string]string) DeleteOption {
	return DeleteOption(cinderowned.DeleteHeaders(values))
}

// ListOpts carries headers and the Resource.list max_items and paginated
// knobs. Python default_types() takes no query, so none is offered.
type ListOpts = cinderowned.ListOpts
type ListOption func(*ListOpts) error

func WithListMaxItems(count int) ListOption   { return ListOption(cinderowned.ListMaxItems(count)) }
func WithListPaginated(value bool) ListOption { return ListOption(cinderowned.ListPaginated(value)) }
func WithListHeaders(values map[string]string) ListOption {
	return ListOption(cinderowned.ListHeaders(values))
}
