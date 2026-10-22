// Package stackcloud ports the Python cloud-layer stack workflows of
// openstack/cloud/_orchestration.py (search_stacks, create_stack,
// update_stack and delete_stack) together with the event polling helper of
// openstack/orchestration/util/event_utils.py. See README.md.
package stackcloud

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/internal/cindervolume"
)

// viewField is one Python Stack attribute in Resource.to_dict order.
type viewField struct {
	attr, wire string
	conv       cindervolume.Conversion
	absent     string
}

// stackViewFields lists the Body and Computed attributes that Stack.to_dict
// returns, in class definition order followed by the inherited Resource id
// and location. The stack_name URI attribute is not part of to_dict, and the
// inherited Resource.name is shadowed by Stack.name (wire key stack_name).
var stackViewFields = [...]viewField{
	{"added", "added", cindervolume.Unchanged, ""},
	{"capabilities", "capabilities", cindervolume.Unchanged, ""},
	{"created_at", "creation_time", cindervolume.Unchanged, ""},
	{"description", "description", cindervolume.Unchanged, ""},
	{"deleted", "deleted", cindervolume.List, ""},
	{"deleted_at", "deletion_time", cindervolume.Unchanged, ""},
	{"environment", "environment", cindervolume.Unchanged, ""},
	{"environment_files", "environment_files", cindervolume.List, ""},
	{"files", "files", cindervolume.Dict, ""},
	{"files_container", "files_container", cindervolume.Unchanged, ""},
	{"is_rollback_disabled", "disable_rollback", cindervolume.Boolean, ""},
	{"links", "links", cindervolume.Unchanged, ""},
	{"name", "stack_name", cindervolume.Unchanged, ""},
	{"notification_topics", "notification_topics", cindervolume.Unchanged, ""},
	{"outputs", "outputs", cindervolume.Unchanged, ""},
	{"owner_id", "stack_owner", cindervolume.Unchanged, ""},
	{"parameters", "parameters", cindervolume.Dict, ""},
	{"parent_id", "parent", cindervolume.Unchanged, ""},
	{"replaced", "replaced", cindervolume.Unchanged, ""},
	{"status", "stack_status", cindervolume.Unchanged, ""},
	{"status_reason", "stack_status_reason", cindervolume.Unchanged, ""},
	{"tags", "tags", cindervolume.List, "[]"},
	{"template", "template", cindervolume.Dict, ""},
	{"template_description", "template_description", cindervolume.Unchanged, ""},
	{"template_url", "template_url", cindervolume.Unchanged, ""},
	{"timeout_mins", "timeout_mins", cindervolume.Unchanged, ""},
	{"unchanged", "unchanged", cindervolume.Unchanged, ""},
	{"updated", "updated", cindervolume.Unchanged, ""},
	{"updated_at", "updated_time", cindervolume.Unchanged, ""},
	{"user_project_id", "stack_user_project_id", cindervolume.Unchanged, ""},
	{"id", "id", cindervolume.Unchanged, ""},
}

// StackView builds the Python Stack.to_dict view that _utils._filter_list
// inspects: attribute names (status, name, owner_id, ...), declared type
// conversions, tags defaulting to [] when absent, unknown response keys
// dropped and location null (it is computed from the Python session). The
// input must be one stack JSON object as Heat returned it.
func StackView(raw json.RawMessage) (json.RawMessage, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, fmt.Errorf("stack must be a JSON object")
	}
	var wire map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &wire); err != nil {
		return nil, err
	}
	return stackViewFromFields(wire)
}

func stackViewFromFields(wire map[string]json.RawMessage) (json.RawMessage, error) {
	var view bytes.Buffer
	view.WriteByte('{')
	for i, field := range stackViewFields {
		if i > 0 {
			view.WriteByte(',')
		}
		key, _ := json.Marshal(field.attr)
		view.Write(key)
		view.WriteByte(':')
		raw, ok := wire[field.wire]
		if !ok {
			if field.absent != "" {
				view.WriteString(field.absent)
			} else {
				view.WriteString("null")
			}
			continue
		}
		converted, err := cindervolume.Convert(raw, field.conv)
		if err != nil {
			return nil, fmt.Errorf("stack field %s: %w", field.wire, err)
		}
		view.Write(converted)
	}
	view.WriteString(`,"location":null}`)
	return json.RawMessage(view.Bytes()), nil
}
