package images

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// TaskRecordObservation holds actual discovery evidence separately from a task.
type TaskRecordObservation struct {
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
}

// ImageTaskRecord separates the declared Source ImageTasks view from the row
// and whole-page receipt. ImageID is the fixed requested parent, not row data.
type ImageTaskRecord struct {
	ImageID        string
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	Microversion   string
	Discovery      []TaskRecordObservation
	fields         map[string]json.RawMessage
}

// Source ImageTasks has 14 untyped Body fields plus inherited id/name. It has no
// Header or URI descriptors: image_id is a Body field and default local filter.
var taskRecordFields = []string{"type", "status", "owner", "expires_at", "created_at", "updated_at", "deleted_at", "deleted", "image_id", "request_id", "user_id", "input", "result", "message", "id", "name"}

func taskRecordInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func taskRecordClone(fields map[string]json.RawMessage) map[string]json.RawMessage {
	out := make(map[string]json.RawMessage, len(fields))
	for k, v := range fields {
		out[k] = bytes.Clone(v)
	}
	return out
}
func taskRecordField(key string) bool {
	for _, name := range taskRecordFields {
		if name == key {
			return true
		}
	}
	return false
}
func (value *ImageTaskRecord) UnmarshalJSON(raw []byte) error {
	var wire resource.RawResource
	if err := json.Unmarshal(raw, &wire); err != nil {
		return err
	}
	members, err := cloudfilter.ObjectMembers(raw)
	if err != nil {
		return err
	}
	fields := make(map[string]json.RawMessage)
	for _, member := range members {
		switch member.Key {
		case "cls", "connection", "microversion", "_synchronized":
			return taskRecordInvalid("Source existing constructor collides with %q", member.Key)
		case "self":
			continue
		}
		if taskRecordField(member.Key) {
			fields[member.Key] = bytes.Clone(member.Value)
		}
	}
	value.Wire = wire.Clone()
	value.fields = fields
	return nil
}
func taskRecordMetadata(v *ImageTaskRecord) *resource.Metadata {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	return &v.Wire.Metadata
}
func taskRecordProject(v *ImageTaskRecord, location json.RawMessage) error {
	body := make(map[string]json.RawMessage, len(taskRecordFields)+1)
	for _, key := range taskRecordFields {
		value, err := resource.BodyRecordField(v.fields, key, resource.BodyFieldJSON)
		if err != nil {
			return err
		}
		body[key] = value
	}
	body["location"] = bytes.Clone(location)
	// The Source constructor retains supplied computed values only when Body,
	// Header and URI are all empty. Such rows fail the mandatory parent filter,
	// but the descriptor view still follows that constructor condition.
	if len(v.fields) == 0 && v.Wire != nil {
		if raw, ok := v.Wire.Body["location"]; ok {
			body["location"] = bytes.Clone(raw)
		}
	}
	v.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: body, Header: v.Header.Clone(), StatusCode: v.StatusCode}}
	return nil
}
func taskRecordMarker(v *ImageTaskRecord) (string, error) {
	if v == nil || v.Wire == nil {
		return "", taskRecordInvalid("actual task id required for marker continuation")
	}
	raw := v.Wire.Body["id"]
	if raw == nil {
		return "", taskRecordInvalid("actual task id required for marker continuation")
	}
	values, err := cloudfilter.RequestQueryValues(raw)
	if err != nil || len(values) != 1 || strings.TrimSpace(values[0]) == "" {
		return "", errors.Join(taskRecordInvalid("marker must have one nonempty actual id query value"), err)
	}
	return values[0], nil
}
func taskRecordTruthy(raw json.RawMessage) (bool, error) {
	token := bytes.TrimSpace(raw)
	if len(token) > 0 && (token[0] == '-' || token[0] >= '0' && token[0] <= '9') && strings.ContainsAny(string(token), ".eE") {
		value, err := strconv.ParseFloat(string(token), 64)
		if err != nil && !errors.Is(err, strconv.ErrRange) {
			return false, err
		}
		return value != 0, nil
	}
	return cloudfilter.PythonTruthy(raw)
}
func taskRecordLocalMatch(actual, filter json.RawMessage) (bool, error) {
	if actual == nil {
		actual = json.RawMessage("null")
	}
	if bytes.HasPrefix(bytes.TrimSpace(filter), []byte("{")) {
		truth, err := taskRecordTruthy(actual)
		if err != nil || !truth {
			return false, err
		}
		wanted, err := cloudfilter.ObjectMembers(filter)
		if err != nil {
			return false, err
		}
		if len(wanted) == 0 {
			return true, nil
		}
		members, err := cloudfilter.ObjectMembers(actual)
		if err != nil {
			return false, taskRecordInvalid("truthy local filter value does not provide mapping.get")
		}
		fields := make(map[string]json.RawMessage)
		for _, member := range members {
			fields[member.Key] = member.Value
		}
		for _, member := range wanted {
			same, err := taskRecordLocalMatch(fields[member.Key], member.Value)
			if err != nil || !same {
				return same, err
			}
		}
		return true, nil
	}
	return jsonfilter.EqualPythonJSON(actual, filter)
}
