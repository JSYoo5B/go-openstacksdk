package actions

import "gophercloudsdk/internal/senlin"

// These are the pinned Action Body fields and their Python attribute aliases.
// The Go model's additional data field is not a pinned Body filter.
var filterSpec = senlin.BodyFilterSpec{
	Namespace: "actions.local_filters",
	Fields: map[string]string{
		"id": "id", "name": "name", "target": "target", "target_id": "target",
		"action": "action", "cause": "cause", "owner": "owner", "owner_id": "owner",
		"user": "user", "user_id": "user", "project": "project", "project_id": "project",
		"domain": "domain", "domain_id": "domain", "interval": "interval",
		"start_time": "start_time", "start_at": "start_time", "end_time": "end_time", "end_at": "end_time",
		"timeout": "timeout", "status": "status", "status_reason": "status_reason",
		"inputs": "inputs", "outputs": "outputs", "depends_on": "depends_on", "depended_by": "depended_by",
		"created_at": "created_at", "updated_at": "updated_at", "cluster_id": "cluster_id",
	},
}

// WithListFilter snapshots a known Body value and filters it locally without a
// wire query. Object filters match recursive subsets; arrays compare completely
// and in order. JSON numbers compare exactly, and booleans remain distinct.
// Query-capable fields may also be selected explicitly for local filtering.
func WithListFilter(key string, value any) ListOption {
	return senlin.WithBodyFilter[ListOpts](filterSpec, key, value)
}
