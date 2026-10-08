// Package cloudsnapshot owns the ordinary Cinder snapshot cloud view and reads.
package cloudsnapshot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type descriptor struct {
	attribute, wire string
	conversion      byte
}

var descriptors = [...]descriptor{
	{"consumes_quota", "consumes_quota", 0},
	{"created_at", "created_at", 0},
	{"description", "description", 0},
	{"group_snapshot_id", "group_snapshot_id", 0},
	{"is_forced", "force", 'b'},
	{"progress", "os-extended-snapshot-attributes:progress", 0},
	{"project_id", "os-extended-snapshot-attributes:project_id", 0},
	{"size", "size", 'i'},
	{"status", "status", 0},
	{"updated_at", "updated_at", 0},
	{"user_id", "user_id", 0},
	{"volume_id", "volume_id", 0},
	{"id", "id", 0},
	{"name", "name", 0},
	{"metadata", "metadata", 'd'},
}

// locationError is an owned policy failure, rather than a new HTTP error.
// Readers retain the admitted page separately and never borrow its status.
type locationError struct{ cause error }

func (e *locationError) Error() string { return e.cause.Error() }
func (e *locationError) Unwrap() error { return e.cause }

// normalize follows the parsed dictionary order and terminal constructor
// to_dict call. All descriptors are consumed before local filtering or find.
// A member's seeded logical ID never creates a field in its raw response.
func normalize(row json.RawMessage, seed *string, list bool, location resource.CloudLocation) (json.RawMessage, bool, error) {
	members, err := cloudfilter.ObjectMembers(row)
	if err != nil {
		return nil, false, err
	}
	var selected [len(descriptors)]json.RawMessage
	for _, member := range members {
		if list {
			switch member.Key {
			case "connection", "microversion", "_synchronized":
				return nil, false, fmt.Errorf("snapshot list constructor collides with %q", member.Key)
			}
		}
		for i, field := range descriptors {
			if member.Key == field.attribute || member.Key == field.wire {
				selected[i] = bytes.Clone(member.Value)
			}
		}
	}
	seeded := seed != nil && selected[12] == nil
	if seeded {
		selected[12], _ = json.Marshal(*seed)
	}
	// Snapshot always has a project descriptor and no zone descriptor. The
	// source recomputes even an otherwise empty connected Resource's location.
	computed, err := location.ForResource(selected[6], nil)
	if err != nil {
		return nil, false, &locationError{err}
	}
	var converted [len(descriptors)]json.RawMessage
	for i, field := range descriptors {
		converted[i], err = convert(selected[i], field.conversion)
		if err != nil {
			return nil, false, fmt.Errorf("snapshot field %q: %w", field.attribute, err)
		}
	}
	var output bytes.Buffer
	output.WriteByte('{')
	for i, field := range descriptors {
		if i > 0 {
			output.WriteByte(',')
		}
		if field.attribute == "metadata" {
			output.WriteString(`"location":`)
			output.Write(computed)
			output.WriteByte(',')
		}
		key, _ := json.Marshal(field.attribute)
		output.Write(key)
		output.WriteByte(':')
		output.Write(converted[i])
	}
	output.WriteByte('}')
	return bytes.Clone(output.Bytes()), seeded, nil
}

func convert(raw json.RawMessage, conversion byte) (json.RawMessage, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return json.RawMessage("null"), nil
	}
	switch conversion {
	case 'i':
		return jsonfilter.DescriptorIntegerJSON(raw)
	case 'd':
		if raw[0] != '{' {
			return json.RawMessage("{}"), nil
		}
	case 'b':
		if bytes.Equal(raw, []byte("true")) || bytes.Equal(raw, []byte("false")) {
			return bytes.Clone(raw), nil
		}
		if raw[0] == '"' {
			var text string
			if err := json.Unmarshal(raw, &text); err != nil {
				return nil, err
			}
			switch strings.ToLower(text) {
			case "true":
				return json.RawMessage("true"), nil
			case "false":
				return json.RawMessage("false"), nil
			}
		}
		return nil, fmt.Errorf("BoolStr expects a boolean or a true/false string")
	}
	return bytes.Clone(raw), nil
}
