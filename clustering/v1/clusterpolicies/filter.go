package clusterpolicies

import "gophercloudsdk/internal/senlin"

// Body filters use the pinned response fields. cluster_id is a URI attribute
// fixed by Scope, and URIClusterID is a Go-only value rather than a Body field.
var filterSpec = senlin.BodyFilterSpec{
	Namespace: "clusterpolicies.local_filters",
	Fields: map[string]string{
		"id": "id", "name": "name", "policy_id": "policy_id", "policy_name": "policy_name",
		"cluster_name": "cluster_name", "policy_type": "policy_type",
		"enabled": "enabled", "is_enabled": "enabled", "data": "data",
	},
}

// WithListFilter snapshots a known response field and compares its raw JSON
// locally, after row validation and pagination accounting. It does not send a
// query or substitute a policy ID for a missing raw binding ID.
func WithListFilter(key string, value any) ListOption {
	return senlin.WithBodyFilter[ListOpts](filterSpec, key, value)
}
