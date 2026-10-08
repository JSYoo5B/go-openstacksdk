package events

import "github.com/JSYoo5B/go-openstacksdk/internal/senlin"

// These are the pinned Event Body fields and their Python attribute aliases.
var filterSpec = senlin.BodyFilterSpec{
	Namespace: "events.local_filters",
	Fields: map[string]string{
		"id": "id", "name": "name", "timestamp": "timestamp", "generated_at": "timestamp",
		"oid": "oid", "obj_id": "oid", "oname": "oname", "obj_name": "oname",
		"otype": "otype", "obj_type": "otype", "cluster_id": "cluster_id", "level": "level",
		"user": "user", "user_id": "user", "project": "project", "project_id": "project",
		"action": "action", "status": "status", "status_reason": "status_reason", "meta_data": "meta_data",
	},
}

// WithListFilter snapshots a known Body value and filters it locally without a
// wire query. Matching uses raw JSON, preserving the original type of level and
// exact numbers in metadata. Query-capable fields may also be filtered locally.
func WithListFilter(key string, value any) ListOption {
	return senlin.WithBodyFilter[ListOpts](filterSpec, key, value)
}
