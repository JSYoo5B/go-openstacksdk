package profiletypes

import "github.com/JSYoo5B/gophercloudsdk/internal/senlin"

func bodyFilterSpec() senlin.BodyFilterSpec {
	return senlin.BodyFilterSpec{Namespace: "profiletypes.local_filters", Fields: map[string]string{
		"id": "id", "name": "name", "schema": "schema", "support_status": "support_status",
	}}
}

// WithListFilter snapshots a known raw response Body field for local filtering.
// Objects match recursive subsets; arrays compare in order and numbers exactly.
// An id filter reads only returned Body id, without falling back to the name.
func WithListFilter(key string, value any) ListOption {
	return senlin.WithBodyFilter[ListOpts](bodyFilterSpec(), key, value)
}
