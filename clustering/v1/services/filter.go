package services

import "gophercloudsdk/internal/senlin"

func bodyFilterSpec() senlin.BodyFilterSpec {
	return senlin.BodyFilterSpec{Namespace: "services.local_filters", Fields: map[string]string{
		"id": "id", "name": "name", "status": "status", "state": "state",
		"binary": "binary", "disabled_reason": "disabled_reason", "host": "host", "updated_at": "updated_at",
	}}
}

// WithListFilter snapshots a known raw response Body field for local filtering.
// Objects match recursive subsets; arrays compare in order and numbers exactly.
// Name and updated_at are retained raw fields even when no typed field exists.
func WithListFilter(key string, value any) ListOption {
	return senlin.WithBodyFilter[ListOpts](bodyFilterSpec(), key, value)
}
