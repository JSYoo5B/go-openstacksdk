package allocationcandidates

import "github.com/JSYoo5B/go-openstacksdk/request"

// WithListQueryValues sends a query key once per value in order, as
// allocation_candidates(member_of1=["agg-4", "agg-5"]) does through requests.
// The values replace the ListOpts serialization of that key, a later
// WithListQuery or WithListQueryValues for the key wins, and no values drop
// only this extension.
func WithListQueryValues(key string, values ...string) ListOption {
	return request.WithQueryValues[ListOpts](key, values...)
}
