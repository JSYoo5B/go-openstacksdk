package users

import "github.com/JSYoo5B/go-openstacksdk/resource"

// projectRecordFilterDescriptor binds the SDK-owned UserProject record, not
// Users.Resources or the native projects.Project model. Its declarations are
// audited by api/openstacksdk/resources/identity/v3/user_project.json and sdkgen's
// fresh pinned-source verification. No unchecked manifest is loaded at runtime.
func projectRecordFilterDescriptor() *resource.FilterDescriptor {
	return &resource.FilterDescriptor{
		Query: map[string]string{
			"domain_id":    "domain_id",
			"is_domain":    "is_domain",
			"name":         "name",
			"parent_id":    "parent_id",
			"is_enabled":   "enabled",
			"tags":         "tags",
			"any_tags":     "tags-any",
			"not_tags":     "not-tags",
			"not_any_tags": "not-tags-any",
			"limit":        "limit",
			"marker":       "marker",
		},
		Body: map[string]string{
			"id":          "id",
			"description": "description",
			"options":     "options",
			"links":       "links",
		},
		Reserved: []string{
			"allow_unknown_params", "base_path", "headers", "jmespath_filters",
			"max_items", "microversion", "paginated", "resource_type", "session",
			// The public proxy supplies this URI argument independently of query.
			"user_id",
		},
	}
}
