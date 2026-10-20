package tapmirrors

import "github.com/JSYoo5B/go-openstacksdk/request"

// WithListQueryValues sends a list query key once per value in order, as
// tap_mirrors(fields=[...], sort_key=[...], sort_dir=[...]) does through
// requests. A later WithListQuery or WithListQueryValues for the key wins, and
// no values drop the key. Neutron keeps the values in its next links.
func WithListQueryValues(key string, values ...string) ListOption {
	return request.WithQueryValues[ListOpts](key, values...)
}
